package server

import (
	"net"
	"sync"
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
	ln       *tcpKeepAliveListener
	Addr     string
	Handler  TcpHandler
	typ      ServerType
	mu       sync.Mutex
	closed   int32
	doneChan chan struct{}
	err      chan error
}

func NewTcpServer(addr string, handler TcpHandler, typ ServerType) *TcpServer {
	return &TcpServer{
		Addr:     addr,
		Handler:  handler,
		typ:      typ,
		doneChan: make(chan struct{}),
		err:      make(chan error, 1),
		closed:   1,
	}
}

func (s *TcpServer) LocalAddr() string { return s.Addr }

func (s *TcpServer) Type() ServerType { return s.typ }

func (s *TcpServer) Error() chan error { return s.err }

func (s *TcpServer) Build() Server { return s }

func (s *TcpServer) Close() error {
	if atomic.LoadInt32(&s.closed) != 0 {
		return ErrServerClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	atomic.StoreInt32(&s.closed, 1)
	close(s.doneChan)

	return s.ln.Close()
}

func (s *TcpServer) Start() {
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
	s.ln = &tcpKeepAliveListener{ln}

	s.err <- nil
	atomic.StoreInt32(&s.closed, 0)
	defer s.Close()
	for {
		rwc, err := ln.Accept()
		if err != nil {
			select {
			case <-s.getDoneChan():
				return
			default:
			}
			continue
		}
		conn := newConn(rwc, nil, s)
		go conn.serve()
	}
}

func (s *TcpServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeTCP(c.conn)
	}
}

func (s *TcpServer) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}
