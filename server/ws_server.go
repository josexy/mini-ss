package server

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/transport"
)

type WsServer struct {
	srv      *http.Server
	Addr     string
	Handler  WsHandler
	opts     *transport.WsOptions
	upgrader *websocket.Upgrader
	running  atomic.Bool
}

func NewWsServer(addr string, handler WsHandler, opts transport.Options) *WsServer {
	return &WsServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*transport.WsOptions),
	}
}

func (s *WsServer) LocalAddr() string { return s.Addr }

func (s *WsServer) Type() ServerType { return Ws }

func (s *WsServer) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrServerStarted
	}
	s.upgrader = &websocket.Upgrader{
		ReadBufferSize:    s.opts.RevBuffer,
		WriteBufferSize:   s.opts.SndBuffer,
		EnableCompression: s.opts.Compress,
		CheckOrigin:       func(r *http.Request) bool { return true },
	}

	laddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		return err
	}

	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}

	var listener net.Listener = &tcpKeepAliveListener{ln}
	tlsConfig, err := s.opts.TlsOptions.GetServerTlsConfig()
	if err != nil {
		return err
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc(s.opts.Path, s.wsUpgrade)
	s.srv = &http.Server{
		Addr:              s.Addr,
		Handler:           serveMux,
		TLSConfig:         tlsConfig, // wss
		ReadHeaderTimeout: 30 * time.Second,
	}
	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	if tlsConfig != nil {
		err = s.srv.ServeTLS(listener, "", "")
	} else {
		err = s.srv.Serve(listener)
	}
	if err != nil && err == http.ErrServerClosed {
		err = nil
	}
	s.running.Store(false)
	return err
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
	conn := newConn(connection.NewWebsocketConn(c), s)
	go conn.serve()
}

func (s *WsServer) Close() error {
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	return s.srv.Shutdown(context.Background())
}

func (s *WsServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeWS(c.conn)
	}
}
