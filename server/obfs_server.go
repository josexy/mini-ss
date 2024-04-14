package server

import (
	"context"
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/transport"
)

type ObfsServer struct {
	srv     *TcpServer
	Addr    string
	Handler ObfsHandler
	opts    *transport.ObfsOptions
}

func NewObfsServer(addr string, handler ObfsHandler, opts transport.Options) *ObfsServer {
	s := &ObfsServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*transport.ObfsOptions),
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
