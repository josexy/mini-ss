package server

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"sync/atomic"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util"
	"github.com/quic-go/quic-go"
)

type QuicServer struct {
	quic.EarlyListener
	Opts     *transport.QuicOptions
	Addr     string
	Handler  QuicHandler
	mu       sync.Mutex
	closed   uint32
	doneChan chan struct{}
	err      chan error
}

func NewQuicServer(addr string, handler QuicHandler, opts transport.Options) *QuicServer {
	return &QuicServer{
		Addr:     addr,
		Handler:  handler,
		doneChan: make(chan struct{}),
		err:      make(chan error, 1),
		Opts:     opts.(*transport.QuicOptions),
		closed:   1,
	}
}

func (s *QuicServer) Error() chan error { return s.err }

func (s *QuicServer) Build() Server { return s }

func (s *QuicServer) LocalAddr() string { return s.Addr }

func (s *QuicServer) Type() ServerType { return Quic }

func (s *QuicServer) Start() {
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: s.Opts.HandshakeIdleTimeout,
		KeepAlivePeriod:      s.Opts.KeepAlivePeriod,
		MaxIdleTimeout:       s.Opts.MaxIdleTimeout,
		Versions: []quic.VersionNumber{
			quic.Version1,
			quic.VersionDraft29,
		},
	}

	var tlsConfig *tls.Config
	if tlsConfig == nil {
		cert, err := util.GenCertificate()
		if err != nil {
			s.err <- err
			return
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	conn, err := net.ListenPacket("udp", s.Addr)
	if err != nil {
		s.err <- err
		return
	}
	ln, err := quic.ListenEarly(conn, transport.TlsConfigQuicALPN(tlsConfig), quicConfig)
	if err != nil {
		s.err <- err
		return
	}
	s.EarlyListener = ln
	defer s.Close()
	s.err <- nil
	atomic.StoreUint32(&s.closed, 0)
	for {
		conn, err := ln.Accept(context.Background())
		if err != nil {
			select {
			case <-s.getDoneChan():
				return
			default:
			}
			continue
		}
		go s.serve(conn)
	}
}

func (s *QuicServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeQUIC(c.conn)
	}
}

func (s *QuicServer) serve(c quic.Connection) {
	for {
		stream, err := c.AcceptStream(context.Background())
		if err != nil {
			return
		}
		conn := newConn(connection.NewQuicConn(stream, c.LocalAddr(), c.RemoteAddr()), s)
		go conn.serve()
	}
}

func (s *QuicServer) Close() error {
	if atomic.LoadUint32(&s.closed) != 0 {
		return ErrServerClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	atomic.StoreUint32(&s.closed, 1)
	close(s.doneChan)
	return s.EarlyListener.Close()
}

func (s *QuicServer) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}
