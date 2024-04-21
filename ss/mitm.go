package ss

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/josexy/mini-ss/util/logger"
)

type fakeHttpResponseWriter struct {
	conn   net.Conn
	bufRW  *bufio.ReadWriter
	header http.Header
}

func newFakeHttpResponseWriter(conn net.Conn) *fakeHttpResponseWriter {
	return &fakeHttpResponseWriter{
		header: make(http.Header),
		conn:   conn,
		bufRW:  bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
}

// Hijack hijack the connection for websocket
func (f *fakeHttpResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return f.conn, f.bufRW, nil
}

// implemented http.ResponseWriter but nothing to do
func (f *fakeHttpResponseWriter) Header() http.Header       { return f.header }
func (f *fakeHttpResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (f *fakeHttpResponseWriter) WriteHeader(int)           {}

func (r *httpReqHandler) initPrivateKeyAndCertPool() {
	r.priKeyPool = cert.NewPriKeyPool(10)
	r.certPool = cert.NewCertPool(r.owner.mitmOpt.fakeCertPool.capacity,
		time.Duration(r.owner.mitmOpt.fakeCertPool.interval)*time.Millisecond,
		time.Duration(r.owner.mitmOpt.fakeCertPool.expireSecond)*time.Millisecond,
	)
}

func (r *httpReqHandler) handleMIMT(ctx context.Context, conn net.Conn) error {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)
	if reqCtx.connMethod {
		// handle https/ws/wss request
		return r.handleConnectMethodRequestAndResponse(ctx, conn)
	} else {
		// handle common http request
		dstConn, err := transport.DialTCP(reqCtx.Hostport())
		if err != nil {
			return err
		}
		defer dstConn.Close()
		return r.handleCommonHTTPRequestAndResponse(ctx, conn, dstConn)
	}
}

func (r *httpReqHandler) getServerResponseCert(ctx context.Context, serverName string) (net.Conn, *tls.Config, error) {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)
	dstConn, err := transport.DialTCP(reqCtx.Hostport())
	if err != nil {
		return nil, nil, err
	}
	tlsConfig := &tls.Config{}
	if serverName != "" {
		tlsConfig.ServerName = serverName
	} else {
		tlsConfig.InsecureSkipVerify = true
	}
	tlsClientConn := tls.Client(dstConn, tlsConfig)
	if err = tlsClientConn.Handshake(); err != nil {
		return nil, nil, err
	}
	// Get server certificate from local cache pool
	if serverCert, err := r.certPool.Get(reqCtx.host); err == nil {
		return tlsClientConn, &tls.Config{Certificates: []tls.Certificate{serverCert}}, nil
	}
	cs := tlsClientConn.ConnectionState()
	var foundCert *x509.Certificate
	for _, cert := range cs.PeerCertificates {
		logger.Logger.Trace("server cert info",
			logx.String("common_name", cert.Subject.CommonName),
			logx.Slice3("country", cert.Subject.Country),
			logx.Slice3("org", cert.Subject.Organization),
			logx.Slice3("org_unit", cert.Subject.OrganizationalUnit),
			logx.Slice3("locality", cert.Subject.Locality),
			logx.Slice3("province", cert.Subject.Province),
			logx.Slice3("dns", cert.DNSNames),
			logx.Slice3("ips", cert.IPAddresses),
			logx.Bool("isca", cert.IsCA),
		)
		if !cert.IsCA {
			foundCert = cert
		}
	}
	if foundCert == nil {
		return nil, nil, errServerCertUnavailable
	}
	// Get private key from local cache pool
	privateKey, err := r.priKeyPool.Get()
	if err != nil {
		return nil, nil, err
	}
	serverCert, err := cert.GenerateCertificate(
		foundCert.Subject, foundCert.DNSNames, foundCert.IPAddresses,
		r.owner.mitmOpt.caCert, r.owner.mitmOpt.caKey, privateKey,
	)
	if err != nil {
		return nil, nil, err
	}
	r.certPool.Add(reqCtx.host, serverCert)
	return tlsClientConn, &tls.Config{Certificates: []tls.Certificate{serverCert}}, nil
}

func (r *httpReqHandler) handleConnectMethodRequestAndResponse(ctx context.Context, conn net.Conn) (err error) {
	// Load ca certificate and key failed
	if r.owner.mitmOpt.caErr != nil {
		return r.owner.mitmOpt.caErr
	}

	// TODO: support common ws request without tls

	var tlsClientConn net.Conn
	tlsServerConn := tls.Server(conn, &tls.Config{
		GetConfigForClient: func(chi *tls.ClientHelloInfo) (tlsConfig *tls.Config, err error) {
			tlsClientConn, tlsConfig, err = r.getServerResponseCert(ctx, chi.ServerName)
			return
		},
	})
	if err = tlsServerConn.Handshake(); err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	defer tlsClientConn.Close()

	ctx, isWsUpgrade, err := r.readHTTPRequestAgainForConnect(ctx, tlsServerConn)
	if err != nil {
		return
	}
	if isWsUpgrade {
		return r.handleCommonWSRequstAndResponse(ctx, tlsServerConn, tlsClientConn)
	}
	return r.handleCommonHTTPRequestAndResponse(ctx, tlsServerConn, tlsClientConn)
}

func (r *httpReqHandler) readHTTPRequestAgainForConnect(ctx context.Context, srcConn net.Conn) (context.Context, bool, error) {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)

	defer func() {
		// Test...
		data, _ := httputil.DumpRequest(reqCtx.request, true)
		fmt.Printf("\n--> dump http request:\n%s\n", string(data))
	}()

	// Read the http request for https/wss via tls tunnel
	if reqCtx.connMethod && reqCtx.request == nil {
		request, err := http.ReadRequest(bufio.NewReader(srcConn))
		if err != nil {
			return ctx, false, err
		}

		// The request url scheme can be either http or https and we don't care
		// Because the inner Dial and DialTLS functions were overwritten and replaced with custom net.Conn
		request.URL.Scheme = "http"
		request.URL.Host = request.Host

		var isWsUpgrade bool
		if request.Header.Get(httpHeaderConnection) == "Upgrade" || request.Header.Get(httpHeaderUpgrade) == "websocket" {
			request.URL.Scheme = "ws"
			isWsUpgrade = true
		}

		reqCtx.request = request
		return context.WithValue(ctx, reqCtxKey, reqCtx), isWsUpgrade, nil
	}

	return ctx, false, nil
}

func (r *httpReqHandler) handleCommonWSRequstAndResponse(ctx context.Context, srcConn, dstConn net.Conn) (err error) {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	upgrader.Subprotocols = []string{reqCtx.request.Header.Get("Sec-WebSocket-Protocol")}
	fakeWriter := newFakeHttpResponseWriter(srcConn)

	// Response to client: HTTP/1.1 101 Switching Protocols
	// Convert net.Conn to websocket.Conn for reading and sending websocket messages
	wsSrcConn, err := upgrader.Upgrade(fakeWriter, reqCtx.request, nil)
	if err != nil {
		return err
	}

	dialer := &websocket.Dialer{
		// override the dial func
		NetDialContext:    func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
		NetDialTLSContext: func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
	}

	// Delete websocket related headers here and re-wrapper them via websocket.Dialer DialContext
	removeRequestHeadersForWebsocket(reqCtx.request.Header)
	// Connect to the real websocket server with the same client request header
	wsDstConn, resp, err := dialer.Dial(reqCtx.request.URL.String(), reqCtx.request.Header)
	if err != nil {
		return err
	}
	resp.Body.Close()

	errCh := make(chan error, 2)
	go func() {
		for {
			msgType, data, err := wsSrcConn.ReadMessage()
			if err != nil {
				errCh <- err
				break
			}
			wsDstConn.WriteMessage(msgType, data)
		}
	}()

	go func() {
		for {
			msgType, data, err := wsDstConn.ReadMessage()
			if err != nil {
				errCh <- err
				break
			}
			wsSrcConn.WriteMessage(msgType, data)
		}
	}()
	err = <-errCh
	return
}

func (r *httpReqHandler) handleCommonHTTPRequestAndResponse(ctx context.Context, srcConn, dstConn net.Conn) (err error) {
	reqCtx := ctx.Value(reqCtxKey).(reqContext)

	// Read the http request for https via tls tunnel
	transport := &http.Transport{
		// override the dial func
		DialContext:    func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
		DialTLSContext: func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
	}
	response, err := transport.RoundTrip(reqCtx.request)
	if err != nil {
		return
	}
	defer response.Body.Close()
	response.Write(srcConn)
	return
}
