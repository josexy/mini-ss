package proxy

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/josexy/mini-ss/util/logger"
)

var (
	errServerCertUnavailable = errors.New("cannot found a available server tls certificate")
	errShortPacket           = errors.New("short packet")
	errMitmDisabled          = errors.New("mitm disabled")
)

var (
	ReqCtxKey   = ReqContextKey{}
	EmptyReqCtx = ReqContext{}
)

type ReqContextKey struct{}

type ReqContext struct {
	// Used for http proxy and sock5 (connect) proxy
	ConnMethod bool
	Host       string
	Port       string
	Addr       string
	Request    *http.Request
}

type MitmHandler interface {
	SetMutableHTTPInterceptor(MutableHTTPInterceptor)
	SetImmutableHTTPInterceptor(ImmutableHTTPInterceptor)
	SetMutableWebsocketInterceptor(MutableWebsocketInterceptor)
	SetImmutableWebsocketInterceptor(ImmutableWebsocketInterceptor)
	HandleMIMT(context.Context, net.Conn) error
	CAPath() string
}

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

type MimtOption struct {
	Enable       bool
	CaPath       string
	KeyPath      string
	FakeCertPool struct {
		Capacity     int
		Interval     int
		ExpireSecond int
	}

	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
}

type mitmHandlerImpl struct {
	mitmOpt       MimtOption
	priKeyPool    *cert.PriKeyPool
	certPool      *cert.CertPool
	immutHttpIntc baseImmutableHTTPInterceptor
	mutHttpIntc   baseMutableHTTPInterceptor
	immutWsIntc   baseImmutableWebsocketInterceptor
	mutWsIntc     baseMutableWebsocketInterceptor
}

func NewMitmHandler(opt MimtOption) (MitmHandler, error) {
	if !opt.Enable {
		return nil, errMitmDisabled
	}
	var err error
	opt.caCert, opt.caKey, err = cert.LoadCACertificate(opt.CaPath, opt.KeyPath)
	if err != nil {
		return nil, err
	}
	handler := &mitmHandlerImpl{
		mitmOpt:    opt,
		priKeyPool: cert.NewPriKeyPool(10),
		certPool: cert.NewCertPool(opt.FakeCertPool.Capacity,
			time.Duration(opt.FakeCertPool.Interval)*time.Millisecond,
			time.Duration(opt.FakeCertPool.ExpireSecond)*time.Millisecond,
		),
	}
	return handler, nil
}

func (r *mitmHandlerImpl) SetMutableHTTPInterceptor(fn MutableHTTPInterceptor) {
	r.immutHttpIntc.fn = nil
	r.mutHttpIntc.fn = fn
}

func (r *mitmHandlerImpl) SetImmutableHTTPInterceptor(fn ImmutableHTTPInterceptor) {
	r.immutHttpIntc.fn = fn
	r.mutHttpIntc.fn = nil
}

func (r *mitmHandlerImpl) SetMutableWebsocketInterceptor(fn MutableWebsocketInterceptor) {
	r.immutWsIntc.fn = nil
	r.mutWsIntc.fn = fn
}

func (r *mitmHandlerImpl) SetImmutableWebsocketInterceptor(fn ImmutableWebsocketInterceptor) {
	r.immutWsIntc.fn = fn
	r.mutWsIntc.fn = nil
}

func (r *mitmHandlerImpl) CAPath() string { return r.mitmOpt.CaPath }

func (r *mitmHandlerImpl) HandleMIMT(ctx context.Context, conn net.Conn) error {
	reqCtx := ctx.Value(ReqCtxKey).(ReqContext)
	if reqCtx.ConnMethod {
		// handle https/ws/wss request
		return r.handleConnectMethodRequestAndResponse(ctx, conn)
	} else {
		// handle common http request
		dstConn, err := transport.DialTCP(reqCtx.Addr)
		if err != nil {
			return err
		}
		defer dstConn.Close()
		return r.handleCommonHTTPRequestAndResponse(ctx, conn, dstConn)
	}
}

func (r *mitmHandlerImpl) getServerResponseCert(ctx context.Context, serverName string) (net.Conn, *tls.Config, error) {
	reqCtx := ctx.Value(ReqCtxKey).(ReqContext)
	dstConn, err := transport.DialTCP(reqCtx.Addr)
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
	if serverCert, err := r.certPool.Get(reqCtx.Host); err == nil {
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
		r.mitmOpt.caCert, r.mitmOpt.caKey, privateKey,
	)
	if err != nil {
		return nil, nil, err
	}
	r.certPool.Add(reqCtx.Host, serverCert)
	return tlsClientConn, &tls.Config{Certificates: []tls.Certificate{serverCert}}, nil
}

func isTLSRequest(data []byte) bool {
	// Check TLS Record Layer: Handshake Protocol
	// data[0]: ContentType: Handshake(0x16)
	// data[1:2]: ProtocolVersion: TLS 1.0(0x0301), TLS 1.1(0x0302), TLS 1.2(0x0303)
	// data[5]: HandshakeType: (Client Hello: 0x1)
	return data[0] == 0x16 && data[1] == 0x3 && (data[2] >= 0x1 && data[2] <= 0x3) && data[5] == 0x1
}

func (r *mitmHandlerImpl) handleConnectMethodRequestAndResponse(ctx context.Context, conn net.Conn) (err error) {
	bufConn := connection.NewBufioConn(conn)
	data, err := bufConn.Peek(6)
	if err != nil {
		return err
	}
	if len(data) < 6 {
		return errShortPacket
	}

	var serverConn, clientConn net.Conn
	// Check if the common http/websocket request with tls
	if isTLSRequest(data) {
		tlsServerConn := tls.Server(bufConn, &tls.Config{
			GetConfigForClient: func(chi *tls.ClientHelloInfo) (tlsConfig *tls.Config, e error) {
				clientConn, tlsConfig, e = r.getServerResponseCert(ctx, chi.ServerName)
				return
			},
		})
		if err = tlsServerConn.Handshake(); err != nil {
			return
		}
		serverConn = tlsServerConn
	} else {
		clientConn, err = transport.DialTCP(ctx.Value(ReqCtxKey).(ReqContext).Addr)
		if err != nil {
			return
		}
		serverConn = bufConn
	}
	defer clientConn.Close()

	ctx, isWsUpgrade, err := r.readHTTPRequestForConnect(ctx, serverConn)
	if err != nil {
		return
	}
	if isWsUpgrade {
		return r.handleCommonWSRequstAndResponse(ctx, serverConn, clientConn)
	}
	return r.handleCommonHTTPRequestAndResponse(ctx, serverConn, clientConn)
}

func (r *mitmHandlerImpl) readHTTPRequestForConnect(ctx context.Context, srcConn net.Conn) (context.Context, bool, error) {
	reqCtx := ctx.Value(ReqCtxKey).(ReqContext)

	// Read the http request for https/wss via tls tunnel
	if reqCtx.ConnMethod && reqCtx.Request == nil {
		request, err := http.ReadRequest(bufio.NewReader(srcConn))
		if err != nil {
			return ctx, false, err
		}

		// The request url scheme can be either http or https and we don't care
		// Because the inner Dial and DialTLS functions were overwritten and replaced with custom net.Conn
		request.URL.Scheme = "http"
		request.URL.Host = request.Host

		var isWsUpgrade bool
		if request.Header.Get(HttpHeaderConnection) == "Upgrade" || request.Header.Get(HttpHeaderUpgrade) == "websocket" {
			request.URL.Scheme = "ws"
			isWsUpgrade = true
		}

		reqCtx.Request = request
		return context.WithValue(ctx, ReqCtxKey, reqCtx), isWsUpgrade, nil
	}

	return ctx, false, nil
}

func (r *mitmHandlerImpl) handleCommonWSRequstAndResponse(ctx context.Context, srcConn, dstConn net.Conn) (err error) {
	reqCtx := ctx.Value(ReqCtxKey).(ReqContext)

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	upgrader.Subprotocols = []string{reqCtx.Request.Header.Get("Sec-WebSocket-Protocol")}
	fakeWriter := newFakeHttpResponseWriter(srcConn)

	// Response to client: HTTP/1.1 101 Switching Protocols
	// Convert net.Conn to websocket.Conn for reading and sending websocket messages
	wsSrcConn, err := upgrader.Upgrade(fakeWriter, reqCtx.Request, nil)
	if err != nil {
		return err
	}

	dialer := &websocket.Dialer{
		// override the dial func
		NetDialContext:    func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
		NetDialTLSContext: func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
	}

	// Delete websocket related headers here and re-wrapper them via websocket.Dialer DialContext
	RemoveWebsocketRequestHeaders(reqCtx.Request.Header)
	// Connect to the real websocket server with the same client request header
	wsDstConn, resp, err := dialer.Dial(reqCtx.Request.URL.String(), reqCtx.Request.Header)
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
			if r.immutWsIntc.fn != nil {
				r.immutWsIntc.InvokeInterceptor(Send, reqCtx.Request, msgType, data, WebsocketDelegatedInvokerFunc(wsDstConn.WriteMessage))
			} else if r.mutWsIntc.fn != nil {
				r.mutWsIntc.InvokeInterceptor(Send, reqCtx.Request, msgType, data, WebsocketDelegatedInvokerFunc(wsDstConn.WriteMessage))
			} else {
				wsDstConn.WriteMessage(msgType, data)
			}
		}
	}()

	go func() {
		for {
			msgType, data, err := wsDstConn.ReadMessage()
			if err != nil {
				errCh <- err
				break
			}
			if r.immutWsIntc.fn != nil {
				r.immutWsIntc.InvokeInterceptor(Receive, reqCtx.Request, msgType, data, WebsocketDelegatedInvokerFunc(wsSrcConn.WriteMessage))
			} else if r.mutWsIntc.fn != nil {
				r.mutWsIntc.InvokeInterceptor(Receive, reqCtx.Request, msgType, data, WebsocketDelegatedInvokerFunc(wsSrcConn.WriteMessage))
			} else {
				wsSrcConn.WriteMessage(msgType, data)
			}
		}
	}()
	err = <-errCh
	return
}

func (r *mitmHandlerImpl) handleCommonHTTPRequestAndResponse(ctx context.Context, srcConn, dstConn net.Conn) (err error) {
	reqCtx := ctx.Value(ReqCtxKey).(ReqContext)

	// Read the http request for https via tls tunnel
	transport := &http.Transport{
		// override the dial func
		DialContext:    func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
		DialTLSContext: func(context.Context, string, string) (net.Conn, error) { return dstConn, nil },
	}
	var response *http.Response
	// Only one http interceptor will be invoked
	if r.immutHttpIntc.fn != nil {
		response, err = r.immutHttpIntc.InvokeInterceptor(reqCtx.Request, HTTPDelegatedInvokerFunc(transport.RoundTrip))
	} else if r.mutHttpIntc.fn != nil {
		response, err = r.mutHttpIntc.InvokeInterceptor(reqCtx.Request, HTTPDelegatedInvokerFunc(transport.RoundTrip))
	} else {
		response, err = transport.RoundTrip(reqCtx.Request)
	}
	if err != nil {
		return
	}
	defer response.Body.Close()
	response.Write(srcConn)
	return
}
