package server

import (
	"context"
	"crypto/tls"
	"net"
	"sync/atomic"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/quic-go/quic-go"
)

type QuicServer struct {
	*quic.EarlyListener
	Opts    *transport.QuicOptions
	Addr    string
	Handler QuicHandler
	running atomic.Bool
}

func NewQuicServer(addr string, handler QuicHandler, opts transport.Options) *QuicServer {
	return &QuicServer{
		Addr:    addr,
		Handler: handler,
		Opts:    opts.(*transport.QuicOptions),
	}
}

func (s *QuicServer) LocalAddr() string { return s.Addr }

func (s *QuicServer) Type() ServerType { return Quic }

func (s *QuicServer) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrServerStarted
	}
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: s.Opts.HandshakeIdleTimeout,
		KeepAlivePeriod:      s.Opts.KeepAlivePeriod,
		MaxIdleTimeout:       s.Opts.MaxIdleTimeout,
		Versions: []quic.VersionNumber{
			quic.Version1,
			quic.Version2,
		},
	}

	var tlsConfig *tls.Config
	if tlsConfig == nil {
		cert, err := cert.GenCertificate()
		if err != nil {
			return err
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	conn, err := net.ListenPacket("udp", s.Addr)
	if err != nil {
		return err
	}
	ln, err := quic.ListenEarly(conn, transport.TlsConfigQuicALPN(tlsConfig), quicConfig)
	if err != nil {
		return err
	}
	s.EarlyListener = ln
	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	for {
		conn, err := ln.Accept(context.Background())
		if err != nil {
			if !s.running.Load() {
				break
			}
			continue
		}
		go s.serve(conn)
	}
	return nil
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
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	return s.EarlyListener.Close()
}
