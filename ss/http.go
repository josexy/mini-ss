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
	"github.com/josexy/mini-ss/proxy"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/util/logger"
)

var (
	errAuthFailed         = errors.New("http-proxy: user authentication failed")
	errHomeAccessed       = errors.New("http-proxy: home accessed")
	errCACertFileAccessed = errors.New("http-proxy: ca cert file accessed")
)

var (
	proxyAgent            = "mini-ss/1.0"
	caCertFileRequestUrl  = "/cacert"
	connectionEstablished = []byte("HTTP/1.1 200 Connection Established\r\nProxy-agent: " + proxyAgent + "\r\n\r\n")
)

type httpReqHandler struct {
	owner    *httpProxyServer
	httpAuth *Auth
}

func newHttpReqHandler(auth *Auth, owner *httpProxyServer) *httpReqHandler {
	return &httpReqHandler{
		httpAuth: auth,
		owner:    owner,
	}
}

func (r *httpReqHandler) ReadRequest(conn net.Conn) (proxy.ReqContext, net.Conn, error) {
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		return proxy.EmptyReqCtx, conn, err
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
	if req.Method == http.MethodConnect || req.Header.Get(proxy.HttpHeaderProxyConnection) != "" {
		return nil
	}
	if r.owner.mitmHandler != nil && req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, caCertFileRequestUrl) {
		resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
		resp.Header.Add(proxy.HttpHeaderContentType, "application/octet-stream")
		resp.Header.Add(proxy.HttpHeaderConnection, "close")
		caFp, err := os.Open(r.owner.mitmHandler.CAPath())
		if err != nil {
			return err
		}
		defer caFp.Close()
		resp.Body = caFp
		resp.Write(conn)
		return errCACertFileAccessed
	}
	resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
	resp.Header.Add(proxy.HttpHeaderConnection, "close")
	resp.Write(conn)
	return errHomeAccessed
}

func (r *httpReqHandler) readRequest(conn net.Conn, req *http.Request) (proxy.ReqContext, net.Conn, error) {
	if err := r.handleHomeAccess(conn, req); err != nil {
		return proxy.EmptyReqCtx, nil, err
	}

	host, port := r.parseHostPort(req)

	if r.owner.mitmHandler == nil && !rule.MatchRuler.Match(&host) {
		return proxy.EmptyReqCtx, nil, constant.ErrRuleMatchDropped
	}

	proxyAuth := req.Header.Get(proxy.HttpHeaderProxyAuthorization)
	var username, password string
	if proxyAuth != "" {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
		if err != nil {
			return proxy.EmptyReqCtx, conn, err
		}
		username, password, _ = strings.Cut(string(data), ":")
	}
	if r.httpAuth != nil && !r.httpAuth.Validate(username, password) {
		errResp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusProxyAuthRequired}
		errResp.Header.Add(proxy.HttpHeaderProxyAgent, proxyAgent)
		errResp.Header.Add(proxy.HttpHeaderProxyAuthenticate, "Basic realm=\"mini-ss\"")
		errResp.Header.Add(proxy.HttpHeaderConnection, "close")
		errResp.Header.Add(proxy.HttpHeaderProxyConnection, "close")
		errResp.Write(conn)
		return proxy.EmptyReqCtx, conn, errAuthFailed
	}

	reqCtx := proxy.ReqContext{
		ConnMethod: req.Method == http.MethodConnect,
		Host:       host,
		Port:       port,
		Addr:       net.JoinHostPort(host, port),
	}
	if req.Method == http.MethodConnect {
		// https: CONNECT www.example.com:443 HTTP/1.1
		// NOTE: ws/wss alos is CONNECT method
		conn.Write(connectionEstablished)
	} else {
		// http: GET/POST/... http://www.example.com/ HTTP/1.1
		proxy.RemoveHopByHopRequestHeaders(req.Header)
		reqCtx.Request = req
	}
	return reqCtx, conn, nil
}

type httpProxyServer struct {
	server.Server
	mitmHandler proxy.MitmHandler
	handler     *httpReqHandler
	pool        *bufferpool.BufferPool
}

func newHttpProxyServer(addr string, httpAuth *Auth) *httpProxyServer {
	hp := &httpProxyServer{}
	hp.pool = bufferpool.NewBytesBufferPool()
	hp.handler = newHttpReqHandler(httpAuth, hp)
	hp.Server = server.NewTcpServer(addr, hp, server.Http)
	return hp
}

func (hp *httpProxyServer) WithMitmMode(opt proxy.MimtOption) *httpProxyServer {
	var err error
	hp.mitmHandler, err = proxy.NewMitmHandler(opt)
	if err != nil {
		logger.Logger.ErrorBy(err)
	}
	if hp.mitmHandler != nil {
		hp.mitmHandler.SetMutableHTTPInterceptor(func(req *http.Request, invoker proxy.HTTPDelegatedInvoker) (*http.Response, error) {
			rsp, err := invoker.Invoke(req)
			if err != nil {
				return rsp, err
			}
			logger.Logger.Debug("http interceptor",
				logx.String("method", req.Method),
				logx.String("host", req.Host),
				logx.String("path", req.URL.Path),
				logx.String("content-type", req.Header.Get("Content-Type")),
				logx.Int("status", rsp.StatusCode),
				logx.Int64("size", rsp.ContentLength),
			)
			return rsp, err
		})
		hp.mitmHandler.SetMutableWebsocketInterceptor(func(dir proxy.WSDirection, req *http.Request,
			msgType int, data []byte, invoker proxy.WebsocketDelegatedInvoker) error {
			direction := "Send"
			if dir == proxy.Receive {
				direction = "Receive"
			}
			logger.Logger.Debug("ws interceptor",
				logx.String("direction", direction),
				logx.String("host", req.Host),
				logx.String("path", req.URL.Path),
				logx.Int("msg-type", msgType),
				logx.Int("data-size", len(data)),
				logx.String("data", string(data)),
			)
			return invoker.Invoke(msgType, data)
		})
	}
	return hp
}

func (hp *httpProxyServer) ServeTCP(conn net.Conn) {
	// read the request and resolve the target host address
	var reqCtx proxy.ReqContext
	var err error
	if reqCtx, conn, err = hp.handler.ReadRequest(conn); err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	// TODO: in mitm mode, the client doesn't relay the data to remote ss server via transport
	if hp.mitmHandler != nil {
		ctx := context.WithValue(context.Background(), proxy.ReqCtxKey, reqCtx)
		if err = hp.mitmHandler.HandleMIMT(ctx, conn); err != nil {
			logger.Logger.ErrorBy(err)
		}
		return
	}

	// convert HTTP request and body to bytes buffer
	if !reqCtx.ConnMethod && reqCtx.Request != nil {
		rbuf := hp.pool.GetBytesBuffer()
		defer hp.pool.PutBytesBuffer(rbuf)
		reqCtx.Request.Write(rbuf)
		reqCtx.Request.Body.Close()
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
			Dst:     reqCtx.Addr,
			Network: "TCP",
			Type:    "HTTP",
			Proxy:   proxy,
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
		})
		// defer statistic.DefaultManager.Remove(tcpTracker)
		conn = tcpTracker
	}
	if err = selector.ProxySelector.Select(proxy)(conn, reqCtx.Addr); err != nil {
		logger.Logger.ErrorBy(err)
	}
}
