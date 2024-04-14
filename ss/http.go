package ss

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/sticky"
	"github.com/josexy/mini-ss/util/logger"
)

var (
	errBadRequest   = errors.New("http-proxy: bad request url scheme")
	errAuthFailed   = errors.New("http-proxy: user authentication failed")
	errHomeAccessed = errors.New("http-proxy: home accessed")
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
		"Proxy-Connection",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}
)

type reqContext struct {
	isHttps bool
	target  string
}

type httpReqHandler struct {
	pool     *bufferpool.BufferPool
	reqCtx   reqContext
	owner    *httpProxyServer
	httpAuth *Auth
}

func newHttpReqHandler(auth *Auth, owner *httpProxyServer) *httpReqHandler {
	return &httpReqHandler{
		httpAuth: auth,
		owner:    owner,
		pool:     bufferpool.NewBytesBufferPool(),
	}
}

func (r *httpReqHandler) ReadRequest(relayer net.Conn) (net.Conn, error) {
	rbuf := r.pool.GetBytesBuffer()
	defer r.pool.PutBytesBuffer(rbuf)

	req, err := http.ReadRequest(bufio.NewReader(io.TeeReader(relayer, rbuf)))
	if err != nil {
		return relayer, err
	}
	return r.readRequest(relayer, req, rbuf)
}

func (r *httpReqHandler) parseHostPort(req *http.Request) (host, port string) {
	var target string
	if req.Method != http.MethodConnect {
		target = req.Host
	} else {
		target = req.RequestURI
	}
	host, port, err := net.SplitHostPort(target)
	if err != nil || port == "" {
		host = target
		if req.Method != http.MethodConnect {
			port = "80"
		}
		// ipv6
		if host[0] == '[' {
			host = target[1 : len(host)-1]
		}
	}
	return
}

func (r *httpReqHandler) handleHomeAccess(conn net.Conn, req *http.Request) error {
	if req.Header.Get("Proxy-Connection") != "" {
		return nil
	}
	resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
	resp.Header.Add("Connection", "close")
	resp.Write(conn)
	return errHomeAccessed
}

func (r *httpReqHandler) readRequest(conn net.Conn, req *http.Request, rbuf *bytes.Buffer) (net.Conn, error) {
	if err := r.handleHomeAccess(conn, req); err != nil {
		return nil, err
	}

	host, port := r.parseHostPort(req)

	logger.Logger.Trace("read request",
		logx.String("method", req.Method),
		logx.String("url", req.URL.String()),
		logx.String("host", host),
		logx.String("port", port),
		logx.Any("header", req.Header),
	)

	if !rule.MatchRuler.Match(&host) {
		return nil, constant.ErrRuleMatchDropped
	}

	proxyAuth := req.Header.Get("Proxy-Authorization")
	var username, password string
	if proxyAuth != "" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
		if err != nil {
			return conn, err
		}
		username, password, _ = strings.Cut(string(data), ":")
	}
	if r.httpAuth != nil && !r.httpAuth.Validate(username, password) {
		errResp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusProxyAuthRequired}
		errResp.Header.Add("Proxy-Agent", proxyAgent)
		errResp.Header.Add("Proxy-Authenticate", "Basic realm=\"mini-ss\"")
		errResp.Header.Add("Connection", "close")
		errResp.Header.Add("Proxy-Connection", "close")
		errResp.Write(conn)
		return conn, errAuthFailed
	}

	if req.Method == http.MethodConnect {
		// https
		// CONNECT www.example.com:443 HTTP/1.1
		conn.Write(connectionEstablished)
	} else {
		// http
		// GET/POST/... http://www.example.com/ HTTP/1.1
		newReq, err := http.ReadRequest(bufio.NewReader(rbuf))
		if err != nil {
			return conn, err
		}
		// delete unnecessary hop-by-hop headers
		delHopReqHeaders(newReq.Header)
		reqBuf := bytes.NewBuffer(make([]byte, 0, 4096))
		newReq.Write(reqBuf)
		newReq.Body.Close()
		conn = sticky.NewSharedReader(reqBuf, conn)
	}
	r.reqCtx = reqContext{
		isHttps: req.Method == http.MethodConnect,
		target:  net.JoinHostPort(host, port),
	}
	return conn, nil
}

func delHopReqHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

type httpProxyServer struct {
	server.Server
	handler *httpReqHandler
	mitmOpt mimtOption
}

func newHttpProxyServer(addr string, httpAuth *Auth) *httpProxyServer {
	hp := &httpProxyServer{}
	hp.handler = newHttpReqHandler(httpAuth, hp)
	hp.Server = server.NewTcpServer(addr, hp, server.Http)
	return hp
}

func (hp *httpProxyServer) WithMitmMode(opt mimtOption) *httpProxyServer {
	hp.mitmOpt = opt
	return hp
}

func (hp *httpProxyServer) ServeTCP(conn net.Conn) {
	// read the request and resolve the target host address
	var err error
	if conn, err = hp.handler.ReadRequest(conn); err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	// check whether mitm mode is enabled
	// TODO: in mitm mode, the client doesn't relay the data to remote ss server via transport
	if hp.mitmOpt.enable {
		if err = hp.handler.handleMIMT(conn); err != nil {
			logger.Logger.ErrorBy(err)
		}
		return
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}
	if statistic.EnableStatistic {
		tcpTracker := statistic.NewTCPTracker(conn, statistic.Context{
			Src:     conn.RemoteAddr().String(),
			Dst:     hp.handler.reqCtx.target,
			Network: "TCP",
			Type:    "HTTP",
			Proxy:   proxy,
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
		})
		// defer statistic.DefaultManager.Remove(tcpTracker)
		conn = tcpTracker
	}
	if err = selector.ProxySelector.Select(proxy)(conn, hp.handler.reqCtx.target); err != nil {
		logger.Logger.ErrorBy(err)
	}
}
