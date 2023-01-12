package server

import (
	"net"
	"sync"
	"sync/atomic"
)

type UdpServer struct {
	Conn
	Addr     string
	Handler  UdpHandler
	typ      ServerType
	mu       sync.Mutex
	closed   int32
	doneChan chan struct{}
	err      chan error
}

func NewUdpServer(addr string, handler UdpHandler, typ ServerType) *UdpServer {
	srv := &UdpServer{
		Addr:     addr,
		Handler:  handler,
		typ:      typ,
		doneChan: make(chan struct{}),
		err:      make(chan error, 1),
		closed:   1,
	}
	return srv
}

func (s *UdpServer) LocalAddr() string { return s.Addr }

func (s *UdpServer) Type() ServerType { return s.typ }

func (s *UdpServer) Error() chan error { return s.err }

func (s *UdpServer) Build() Server { return s }

func (s *UdpServer) Close() error {
	if atomic.LoadInt32(&s.closed) != 0 {
		return ErrServerClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	atomic.StoreInt32(&s.closed, 1)
	close(s.doneChan)
	return s.Conn.close()
}

func (s *UdpServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeUDP(c.packetConn)
	}
}

func (s *UdpServer) Start() {
	defer func() {
		recover()
		s.Close()
	}()

	// listen on unconnected udp server
	conn, err := net.ListenPacket("udp", s.Addr)
	if err != nil {
		s.err <- err
		return
	}
	s.Conn = newConn(nil, conn, s)
	s.err <- nil
	atomic.StoreInt32(&s.closed, 0)
	for {
		select {
		case <-s.getDoneChan():
			// server closed
			return
		default:
		}
		s.Conn.serve()
	}
}

func (s *UdpServer) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doneChan
}
