package server

import (
	"context"
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/options"
)

var _ Server = (*ObfsServer)(nil)

type ObfsServer struct {
	srv     *TcpServer
	Addr    string
	Handler ObfsHandler
	opts    *options.ObfsOptions
}

func NewObfsServer(addr string, handler ObfsHandler, opts options.Options) *ObfsServer {
	s := &ObfsServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*options.ObfsOptions),
	}
	s.srv = NewTcpServer(addr, TcpHandlerFunc(s.serveTCP), Obfs)
	return s
}

func (s *ObfsServer) Start(ctx context.Context) error { return s.srv.Start(ctx) }

func (s *ObfsServer) Close() error { return s.srv.Close() }

func (s *ObfsServer) LocalAddr() string { return s.Addr }

func (s *ObfsServer) Type() ServerType { return s.srv.typ }

func (s *ObfsServer) Serve(*Conn) {}

func (s *ObfsServer) serveTCP(c net.Conn) {
	c = connection.NewObfsConn(c, s.opts.Host, true)
	if s.Handler != nil {
		s.Handler.ServeOBFS(c)
	}
}
