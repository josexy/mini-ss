package ss

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/josexy/mini-ss/util/logger"
)

const (
	httpHeaderContentType            = "Content-Type"
	httpHeaderConnection             = "Connection"
	httpHeaderKeepAlive              = "Keep-Alive"
	httpHeaderProxyAuthenticate      = "Proxy-Authenticate"
	httpHeaderProxyAuthorization     = "Proxy-Authorization"
	httpHeaderProxyConnection        = "Proxy-Connection"
	httpHeaderProxyAgent             = "Proxy-Agent"
	httpHeaderTe                     = "Te"
	httpHeaderTrailers               = "Trailers"
	httpHeaderTransferEncoding       = "Transfer-Encoding"
	httpHeaderUpgrade                = "Upgrade"
	httpHeaderSecWebsocketKey        = "Sec-Websocket-Key"
	httpHeaderSecWebsocketVersion    = "Sec-Websocket-Version"
	httpHeaderSecWebsocketExtensions = "Sec-Websocket-Extensions"
)

var (
	errAuthFailed            = errors.New("http-proxy: user authentication failed")
	errHomeAccessed          = errors.New("http-proxy: home accessed")
	errCACertFileAccessed    = errors.New("http-proxy: ca cert file accessed")
	errServerCertUnavailable = errors.New("cannot found a available server tls certificate")
)

var (
	proxyAgent            = "mini-ss/1.0"
	caCertFileRequestUrl  = "/cacert"
	connectionEstablished = []byte("HTTP/1.1 200 Connection Established\r\nProxy-agent: " + proxyAgent + "\r\n\r\n")

	// Hop-by-hop headers. These are removed when sent to the backend.
	// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
	hopByHopHeaders = []string{
		httpHeaderConnection,
		httpHeaderKeepAlive,
		httpHeaderProxyAuthenticate,
		httpHeaderProxyAuthorization,
		httpHeaderTe,
		httpHeaderTrailers,
		httpHeaderTransferEncoding,
		httpHeaderUpgrade,
		httpHeaderProxyConnection,
	}
	reqCtxKey   = reqContextKey{}
	emptyReqCtx = reqContext{}
)

type reqContextKey struct{}

type reqContext struct {
	connMethod bool
	host       string
	port       string
	request    *http.Request
}

func (r *reqContext) Hostport() string {
	return net.JoinHostPort(r.host, r.port)
}

type httpReqHandler struct {
	owner      *httpProxyServer
	httpAuth   *Auth
	priKeyPool *cert.PriKeyPool // For MITM
	certPool   *cert.CertPool   // For MITM
}

func newHttpReqHandler(auth *Auth, owner *httpProxyServer) *httpReqHandler {
	return &httpReqHandler{
		httpAuth: auth,
		owner:    owner,
	}
}

func (r *httpReqHandler) ReadRequest(conn net.Conn) (reqContext, net.Conn, error) {
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return emptyReqCtx, conn, err
	}
	return r.readRequest(conn, req)
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

// handleHomeAccess only handle GET/POST... common method
func (r *httpReqHandler) handleHomeAccess(conn net.Conn, req *http.Request) error {
	if req.Method == http.MethodConnect || req.Header.Get(httpHeaderProxyConnection) != "" {
		return nil
	}
	if r.owner.mitmOpt.enable && req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, caCertFileRequestUrl) {
		resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
		resp.Header.Add(httpHeaderContentType, "application/octet-stream")
		resp.Header.Add(httpHeaderConnection, "close")
		caFp, err := os.Open(r.owner.mitmOpt.caPath)
		if err != nil {
			return err
		}
		defer caFp.Close()
		resp.Body = caFp
		resp.Write(conn)
		return errCACertFileAccessed
	}
	resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
	resp.Header.Add(httpHeaderConnection, "close")
	resp.Write(conn)
	return errHomeAccessed
}

func (r *httpReqHandler) readRequest(conn net.Conn, req *http.Request) (reqContext, net.Conn, error) {
	if err := r.handleHomeAccess(conn, req); err != nil {
		return emptyReqCtx, nil, err
	}

	host, port := r.parseHostPort(req)

	logger.Logger.Trace("read request",
		logx.String("method", req.Method),
		logx.String("url", req.URL.String()),
		logx.String("host", host),
		logx.String("port", port),
		logx.Any("header", req.Header),
	)

	if !r.owner.mitmOpt.enable && !rule.MatchRuler.Match(&host) {
		return emptyReqCtx, nil, constant.ErrRuleMatchDropped
	}

	proxyAuth := req.Header.Get(httpHeaderProxyAuthorization)
	var username, password string
	if proxyAuth != "" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
		if err != nil {
			return emptyReqCtx, conn, err
		}
		username, password, _ = strings.Cut(string(data), ":")
	}
	if r.httpAuth != nil && !r.httpAuth.Validate(username, password) {
		errResp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusProxyAuthRequired}
		errResp.Header.Add(httpHeaderProxyAgent, proxyAgent)
		errResp.Header.Add(httpHeaderProxyAuthenticate, "Basic realm=\"mini-ss\"")
		errResp.Header.Add(httpHeaderConnection, "close")
		errResp.Header.Add(httpHeaderProxyConnection, "close")
		errResp.Write(conn)
		return emptyReqCtx, conn, errAuthFailed
	}

	reqCtx := reqContext{
		connMethod: req.Method == http.MethodConnect,
		host:       host,
		port:       port,
	}
	if req.Method == http.MethodConnect {
		// https: CONNECT www.example.com:443 HTTP/1.1
		// NOTE: ws/wss alos is CONNECT method
		conn.Write(connectionEstablished)
	} else {
		// http: GET/POST/... http://www.example.com/ HTTP/1.1
		removeHopByHopRequestHeaders(req.Header)
		reqCtx.request = req
	}
	return reqCtx, conn, nil
}

func removeHopByHopRequestHeaders(header http.Header) {
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

func removeRequestHeadersForWebsocket(header http.Header) {
	header.Del(httpHeaderUpgrade)
	header.Del(httpHeaderConnection)
	header.Del(httpHeaderSecWebsocketKey)
	header.Del(httpHeaderSecWebsocketVersion)
	header.Del(httpHeaderSecWebsocketExtensions)
}

type httpProxyServer struct {
	server.Server
	handler *httpReqHandler
	pool    *bufferpool.BufferPool
	mitmOpt mimtOption
}

func newHttpProxyServer(addr string, httpAuth *Auth) *httpProxyServer {
	hp := &httpProxyServer{}
	hp.pool = bufferpool.NewBytesBufferPool()
	hp.handler = newHttpReqHandler(httpAuth, hp)
	hp.Server = server.NewTcpServer(addr, hp, server.Http)
	return hp
}

func (hp *httpProxyServer) WithMitmMode(opt mimtOption) *httpProxyServer {
	opt.caCert, opt.caKey, opt.caErr = cert.LoadCACertificate(opt.caPath, opt.keyPath)
	hp.mitmOpt = opt
	hp.handler.initPrivateKeyAndCertPool()
	return hp
}

func (hp *httpProxyServer) ServeTCP(conn net.Conn) {
	// read the request and resolve the target host address
	var reqCtx reqContext
	var err error
	if reqCtx, conn, err = hp.handler.ReadRequest(conn); err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	ctx := context.WithValue(context.Background(), reqCtxKey, reqCtx)
	// check whether mitm mode is enabled
	// TODO: in mitm mode, the client doesn't relay the data to remote ss server via transport
	if hp.mitmOpt.enable {
		if err = hp.handler.handleMIMT(ctx, conn); err != nil {
			logger.Logger.ErrorBy(err)
		}
		return
	}

	// convert HTTP request and body to bytes buffer
	if !reqCtx.connMethod && reqCtx.request != nil {
		rbuf := hp.pool.GetBytesBuffer()
		defer hp.pool.PutBytesBuffer(rbuf)
		reqCtx.request.Write(rbuf)
		reqCtx.request.Body.Close()
		conn = connection.NewConnWithReader(conn, rbuf)
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}
	if statistic.EnableStatistic {
		tcpTracker := statistic.NewTCPTracker(conn, statistic.Context{
			Src:     conn.RemoteAddr().String(),
			Dst:     reqCtx.Hostport(),
			Network: "TCP",
			Type:    "HTTP",
			Proxy:   proxy,
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
		})
		// defer statistic.DefaultManager.Remove(tcpTracker)
		conn = tcpTracker
	}
	if err = selector.ProxySelector.Select(proxy)(conn, reqCtx.Hostport()); err != nil {
		logger.Logger.ErrorBy(err)
	}
}
