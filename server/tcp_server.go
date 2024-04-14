package server

import (
	"context"
	"net"
	"sync/atomic"
	"time"
)

type tcpKeepAliveListener struct{ *net.TCPListener }

func (l *tcpKeepAliveListener) Accept() (net.Conn, error) {
	conn, err := l.TCPListener.AcceptTCP()
	if err != nil {
		return nil, err
	}
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(180 * time.Second)
	return conn, nil
}

type TcpServer struct {
	ln      *tcpKeepAliveListener
	Addr    string
	Handler TcpHandler
	typ     ServerType
	running atomic.Bool
}

func NewTcpServer(addr string, handler TcpHandler, typ ServerType) *TcpServer {
	return &TcpServer{
		Addr:    addr,
		Handler: handler,
		typ:     typ,
	}
}

func (s *TcpServer) LocalAddr() string { return s.Addr }

func (s *TcpServer) Type() ServerType { return s.typ }

func (s *TcpServer) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrServerStarted
	}
	laddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		return err
	}
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}
	s.ln = &tcpKeepAliveListener{ln}

	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	for {
		rwc, err := ln.Accept()
		if err != nil {
			if !s.running.Load() {
				break
			}
			continue
		}
		conn := newConn(rwc, s)
		go conn.serve()
	}
	return nil
}

func (s *TcpServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeTCP(c.conn)
	}
}

func (s *TcpServer) Close() error {
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	return s.ln.Close()
}
