package ss

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/interceptor"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/mitmpgo"
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

var hopByHopHeaders = []string{
	mitmpgo.HttpHeaderConnection,
	mitmpgo.HttpHeaderKeepAlive,
	mitmpgo.HttpHeaderProxyAuthenticate,
	mitmpgo.HttpHeaderProxyAuthorization,
	mitmpgo.HttpHeaderTe,
	mitmpgo.HttpHeaderTrailers,
	mitmpgo.HttpHeaderTransferEncoding,
	mitmpgo.HttpHeaderUpgrade,
	mitmpgo.HttpHeaderProxyConnection,
}

type httpReqContext struct {
	request  *http.Request
	hostport string
}

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

func (r *httpReqHandler) ReadRequest(conn net.Conn) (req *http.Request, hostport string, err error) {
	if req, err = http.ReadRequest(bufio.NewReader(conn)); err != nil {
		return
	}
	if hostport, err = r.handlePreRequest(conn, req); err != nil {
		return
	}
	return req, hostport, nil
}

// handleHomeAccess only handle GET/POST... common method
func (r *httpReqHandler) handleHomeAccess(conn net.Conn, req *http.Request) error {
	// 1. CONNECT xxxx
	// 2. POST http://xxxx
	// 3. POST /xxxx
	if req.Method == http.MethodConnect || req.Header.Get(mitmpgo.HttpHeaderProxyConnection) != "" || req.URL.Scheme != "" {
		return nil
	}
	if r.owner.mitmHandler != nil && req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, caCertFileRequestUrl) {
		resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
		resp.Header.Add(mitmpgo.HttpHeaderContentType, "application/x-x509-ca-cert")
		resp.Header.Add(mitmpgo.HttpHeaderConnection, "close")
		caFp, err := os.Open(r.owner.mitmHandler.CACertPath())
		if err != nil {
			return err
		}
		defer caFp.Close()
		resp.Body = caFp
		resp.Write(conn)
		return errCACertFileAccessed
	}
	resp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusOK}
	resp.Header.Add(mitmpgo.HttpHeaderConnection, "close")
	resp.Write(conn)
	return fmt.Errorf("%s, url: %s", errHomeAccessed, req.URL.String())
}

func (r *httpReqHandler) handlePreRequest(conn net.Conn, req *http.Request) (string, error) {
	if err := r.handleHomeAccess(conn, req); err != nil {
		return "", err
	}

	hostport, _ := mitmpgo.ParseHostPort(req)
	host, _, _ := net.SplitHostPort(hostport)

	if r.owner.mitmHandler == nil && !rule.MatchRuler.Match(&host) {
		return "", rule.ErrRuleMatchDropped
	}

	proxyAuth := req.Header.Get(mitmpgo.HttpHeaderProxyAuthorization)
	var username, password string
	if proxyAuth != "" {
		if strings.Contains(proxyAuth, "Basic") && len(proxyAuth) >= 6 {
			data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(proxyAuth, "Basic "))
			if err != nil {
				return "", err
			}
			username, password, _ = strings.Cut(string(data), ":")
		}
	}
	if r.httpAuth != nil && !r.httpAuth.Validate(username, password) {
		errResp := &http.Response{ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), StatusCode: http.StatusProxyAuthRequired}
		errResp.Header.Add(mitmpgo.HttpHeaderProxyAgent, proxyAgent)
		errResp.Header.Add(mitmpgo.HttpHeaderProxyAuthenticate, "Basic realm=\"mini-ss\"")
		errResp.Header.Add(mitmpgo.HttpHeaderConnection, "close")
		errResp.Header.Add(mitmpgo.HttpHeaderProxyConnection, "close")
		errResp.Write(conn)
		return "", errAuthFailed
	}

	if req.Method == http.MethodConnect {
		// https: CONNECT www.example.com:443 HTTP/1.1
		// NOTE: ws/wss alos is CONNECT method
		conn.Write(connectionEstablished)
	} else {
		// http: GET/POST/... http://www.example.com/ HTTP/1.1
		for _, h := range hopByHopHeaders {
			req.Header.Del(h)
		}
	}
	return hostport, nil
}

type httpProxyServer struct {
	server.Server
	mitmHandler mitmpgo.MitmProxyHandler
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

func (hp *httpProxyServer) WithMitmMode(opts []mitmpgo.Option) *httpProxyServer {
	if len(opts) == 0 {
		return hp
	}
	opts = append(opts,
		mitmpgo.WithErrorHandler(interceptor.ErrHandler),
		mitmpgo.WithHTTPInterceptor(interceptor.HttpInterceptor),
		mitmpgo.WithWebsocketInterceptor(interceptor.WebsocketInterceptor),
	)
	var err error
	hp.mitmHandler, err = mitmpgo.NewMitmProxyHandler(opts...)
	if err != nil {
		logger.Logger.ErrorWith(err)
	}
	return hp
}

func (hp *httpProxyServer) ServeTCP(conn net.Conn) {
	// read the request and resolve the target host address
	req, hostport, err := hp.handler.ReadRequest(conn)
	if err != nil {
		logger.Logger.ErrorWith(err)
		return
	}

	if hp.mitmHandler != nil {
		if req.Method == http.MethodConnect {
			req = nil
		}
		ctx := mitmpgo.AppendToRequestContext(context.Background(), hostport, req)
		if err = hp.mitmHandler.Serve(ctx, conn); err != nil {
			logger.Logger.ErrorWith(err)
		}
		return
	}

	// convert HTTP request and body to bytes buffer
	if req.Method != http.MethodConnect {
		var buf bytes.Buffer
		req.Write(&buf)
		req.Body.Close()
		conn = connection.NewConnWithReader(conn, &buf)
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorWith(err)
		return
	}
	if statistic.EnableStatistic {
		tcpTracker := statistic.NewTCPTracker(conn, statistic.Context{
			Src:     conn.RemoteAddr().String(),
			Dst:     hostport,
			Network: "TCP",
			Type:    "HTTP",
			Proxy:   proxy,
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
		})
		// defer statistic.DefaultManager.Remove(tcpTracker)
		conn = tcpTracker
	}
	if err = selector.ProxySelector.Select(proxy).Invoke(conn, hostport); err != nil {
		logger.Logger.ErrorWith(err)
	}
}
