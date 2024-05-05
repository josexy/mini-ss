package server

import (
	"context"
	"crypto/tls"
	"crypto/x509/pkix"
	"sync/atomic"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/cert"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/quic-go/quic-go"
)

var _ Server = (*QuicServer)(nil)

type QuicServer struct {
	*quic.EarlyListener
	Addr    string
	Handler QuicHandler
	running atomic.Bool
	conns   []quic.EarlyConnection
	opts    *options.QuicOptions
}

func NewQuicServer(addr string, handler QuicHandler, opts options.Options) *QuicServer {
	return &QuicServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*options.QuicOptions),
		conns:   make([]quic.EarlyConnection, 0, 32),
	}
}

func (s *QuicServer) LocalAddr() string { return s.Addr }

func (s *QuicServer) Type() ServerType { return Quic }

func (s *QuicServer) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrServerStarted
	}

	tlsConfig, err := s.opts.TlsOptions.GetServerTlsConfig()
	if err != nil {
		return err
	}
	// Using self-generated tls config
	if tlsConfig == nil {
		privateKey, err := cert.GeneratePrivateKey()
		if err != nil {
			return err
		}
		cert, err := cert.GenerateCertificate(pkix.Name{CommonName: "mini-ss"}, nil, nil, nil, nil, privateKey)
		if err != nil {
			return err
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	conn, err := transport.ListenUDP(ctx, s.Addr)
	if err != nil {
		return err
	}

	ln, err := quic.ListenEarly(conn, transport.TlsConfigQuicALPN(tlsConfig), &quic.Config{
		HandshakeIdleTimeout: s.opts.HandshakeIdleTimeout,
		KeepAlivePeriod:      s.opts.KeepAlivePeriod,
		MaxIdleTimeout:       s.opts.MaxIdleTimeout,
		Versions: []quic.VersionNumber{
			quic.Version1,
			quic.Version2,
		},
	})
	if err != nil {
		return err
	}
	s.EarlyListener = ln
	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	for {
		conn, err := ln.Accept(ctx)
		if err != nil {
			if !s.running.Load() {
				break
			}
			continue
		}
		logger.Logger.Tracef("quic accept connection: %s", conn.RemoteAddr())
		s.conns = append(s.conns, conn)
		go s.acceptStreamForConn(ctx, conn)
	}
	return nil
}

func (s *QuicServer) acceptStreamForConn(ctx context.Context, conn quic.Connection) {
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return
		}
		logger.Logger.Tracef("accept stream: [%s]-[%d]", conn.RemoteAddr(), stream.StreamID())
		go newConn(connection.NewQuicConn(stream, conn.LocalAddr(), conn.RemoteAddr()), s).serve()
	}
}

func (s *QuicServer) Serve(conn *Conn) {
	if s.Handler != nil {
		s.Handler.ServeQUIC(conn)
	}
}

func (s *QuicServer) Close() error {
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	err := s.EarlyListener.Close()
	for _, conn := range s.conns {
		_ = conn.CloseWithError(quic.ApplicationErrorCode(0), "")
	}
	s.conns = nil
	return err
}
