package server

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util"
)

type WsServer struct {
	srv      *http.Server
	Addr     string
	Handler  WsHandler
	opts     *transport.WsOptions
	upgrader *websocket.Upgrader
	err      chan error
}

func NewWsServer(addr string, handler WsHandler, opts transport.Options) *WsServer {
	return &WsServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*transport.WsOptions),
		err:     make(chan error, 1),
	}
}

func (s *WsServer) LocalAddr() string { return s.Addr }

func (s *WsServer) Error() chan error { return s.err }

func (s *WsServer) Build() Server { return s }

func (s *WsServer) Type() ServerType { return Ws }

func (s *WsServer) Start() {
	s.upgrader = &websocket.Upgrader{
		ReadBufferSize:    s.opts.RevBuffer,
		WriteBufferSize:   s.opts.SndBuffer,
		EnableCompression: s.opts.Compress,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}

	laddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		s.err <- err
		return
	}

	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		s.err <- err
		return
	}

	var listener net.Listener = &tcpKeepAliveListener{ln}
	var tlsConfig *tls.Config
	if s.opts.TLS {
		cert, err := util.GenCertificate()
		if err != nil {
			s.err <- err
			return
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		listener = tls.NewListener(listener, tlsConfig)
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(s.opts.Path, s.wsUpgrade)
	s.srv = &http.Server{
		Addr:              s.Addr,
		Handler:           serveMux,
		TLSConfig:         tlsConfig, // wss
		ReadHeaderTimeout: 30 * time.Second,
	}
	s.err <- nil
	s.srv.Serve(listener)
}

func (s *WsServer) wsUpgrade(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if host == "" && r.URL != nil {
		host = r.URL.Host
	}
	if host != s.opts.Host {
		return
	}
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := newConn(transport.NewWebsocketConn(c), s)
	go conn.serve()
}

func (s *WsServer) Close() error {
	return s.srv.Close()
}

func (s *WsServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeWS(c.conn)
	}
}
