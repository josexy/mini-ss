package ss

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/auth"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/socks/constant"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/sticky"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/ordmap"
)

var (
	proxyAgent            = "mini-ss/1.0"
	connectionEstablished = []byte("HTTP/1.1 200 Connection Established\r\nProxy-agent: " + proxyAgent + "\r\n\r\n")

	// Hop-by-hop headers. These are removed when sent to the backend.
	// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
	hopHeaders = []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te", // canonicalized version of "TE"
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
)

type httpReqHandler struct {
	httpAuth *auth.Auth
	DstAddr  string
	*dns.Ruler
}

func newhttpReqHandler(auth *auth.Auth) *httpReqHandler {
	return &httpReqHandler{
		httpAuth: auth,
	}
}

func (r *httpReqHandler) ReadRequest(relayer net.Conn) (net.Conn, error) {
	rbuf := bytes.Buffer{}
	req, err := http.ReadRequest(bufio.NewReader(io.TeeReader(relayer, &rbuf)))
	if err != nil {
		return relayer, err
	}
	return r.readRequest(relayer, req, &rbuf)
}

func (r *httpReqHandler) parseHostPort(req *http.Request) (host, port string) {
	var target string
	if req.Method != http.MethodConnect {
		target = req.Host
		if !strings.Contains(target, ":") {
			target += ":80"
		}
	} else {
		target = req.RequestURI
	}
	host, port, _ = net.SplitHostPort(target)
	return
}

func (r *httpReqHandler) readRequest(conn net.Conn, req *http.Request, rbuf *bytes.Buffer) (net.Conn, error) {
	host, port := r.parseHostPort(req)

	if r.Ruler.Match(&host) {
		return nil, constant.ErrRuleMatchDropped
	}

	// HTTP/1.1
	errResp := &http.Response{
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
	}

	errResp.Header.Add("Proxy-Agent", proxyAgent)

	// connect method only use https/tls
	if req.Method != http.MethodConnect && req.URL.Scheme != "http" {
		errResp.StatusCode = http.StatusBadRequest
		errResp.Write(conn)
		return conn, errors.New("http-proxy: bad request url scheme")
	}

	val := req.Header.Get("Proxy-Authorization")
	var username, password string
	if val != "" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(val, "Basic "))
		if err != nil {
			return conn, err
		}
		username, password, _ = strings.Cut(string(data), ":")
	}
	if r.httpAuth != nil && !r.httpAuth.Validate(username, password) {
		errResp.StatusCode = http.StatusProxyAuthRequired
		errResp.Header.Add("Proxy-Authenticate", "Basic realm=\"mini-ss\"")
		errResp.Header.Add("Connection", "close")
		errResp.Header.Add("Proxy-Connection", "close")
		errResp.Write(conn)
		return conn, errors.New("http-proxy: user auth failed")
	}

	if req.Method != http.MethodConnect {
		// http
		// GET/POST/... http://www.example.com/ HTTP/1.1
		// ss-local parses the http request sent by the client
		newReq, err := http.ReadRequest(bufio.NewReader(rbuf))
		if err != nil {
			return conn, err
		}
		// delete unnecessary hop-by-hop headers
		delHopReqHeaders(newReq.Header)
		newReq.Header.Del("Proxy-Connection")

		reqBuf := bytes.NewBuffer(make([]byte, 0, 1024))
		newReq.Write(reqBuf)
		newReq.Body.Close()
		conn = sticky.NewSharedReader(reqBuf, conn)
	} else {
		// https
		// CONNECT www.example.com:443 HTTP/1.1
		conn.Write(connectionEstablished)
	}
	r.DstAddr = net.JoinHostPort(host, port)
	return conn, nil
}

func delHopReqHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

type httpProxyServer struct {
	server.Server
	addr     string
	handler  *httpReqHandler
	relayers ordmap.OrderedMap
	ruler    *dns.Ruler
}

func newHttpProxyServer(ctx context.Context, addr string, httpAuth *auth.Auth) *httpProxyServer {
	pv := ctx.Value(ctxv.SSLocalContextKey).(*ctxv.ContextPassValue)
	hp := &httpProxyServer{
		ruler:   pv.R,
		addr:    addr,
		handler: newhttpReqHandler(httpAuth),
	}
	hp.handler.Ruler = pv.R
	pv.MAP.Range(func(name, value any) bool {
		v := value.(ctxv.V)
		hp.relayers.Store(name, transport.DstAddrRelayer{
			DstAddr:          v.Addr,
			TCPRelayer:       transport.NewTCPRelayer(constant.TCPSSLocalToSSServer, v.Type, v.Options, nil, v.TcpConnBound),
			TCPDirectRelayer: transport.NewTCPDirectRelayer(),
		})
		return true
	})

	if pv.MAP.Size() == 0 && pv.R.RuleMode == dns.Direct {
		hp.relayers.Store("", transport.DstAddrRelayer{TCPDirectRelayer: transport.NewTCPDirectRelayer()})
	}
	return hp
}

func (hp *httpProxyServer) Build() server.Server {
	hp.Server = server.NewTcpServer(hp.addr, hp, server.Http)
	return hp
}

func (hp *httpProxyServer) ServeTCP(conn net.Conn) {
	// read the request and parse destination host address
	var err error
	if conn, err = hp.handler.ReadRequest(conn); err != nil {
		logx.ErrorBy(err)
		return
	}

	if err = hp.ruler.Select(&hp.relayers, conn, hp.handler.DstAddr); err != nil {
		logx.ErrorBy(err)
	}
}
