package server

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/mux"
	"github.com/josexy/mini-ss/options"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

var _ Server = (*KcpServer)(nil)

type KcpServer struct {
	net.Listener
	Addr       string
	Handler    KcpHandler
	smuxConfig *smux.Config
	once       sync.Once
	opts       *options.KcpOptions
	running    atomic.Bool
	err        chan error
}

func NewKcpServer(addr string, handler KcpHandler, opts options.Options) *KcpServer {
	return &KcpServer{
		Addr:    addr,
		Handler: handler,
		err:     make(chan error, 1),
		opts:    opts.((*options.KcpOptions)),
	}
}

func (s *KcpServer) LocalAddr() string { return s.Addr }

func (s *KcpServer) Type() ServerType { return Kcp }

func (s *KcpServer) Close() error {
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	return s.Listener.Close()
}

func (s *KcpServer) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrServerStarted
	}
	s.opts.Update()

	ln, err := kcp.ListenWithOptions(s.Addr, s.opts.BC, s.opts.DataShard, s.opts.ParityShard)
	if err != nil {
		return err
	}
	s.Listener = ln
	if s.opts.Dscp > 0 {
		ln.SetDSCP(s.opts.Dscp)
	}
	ln.SetReadBuffer(s.opts.SockBuf)
	ln.SetWriteBuffer(s.opts.SockBuf)

	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	for {
		sess, err := ln.AcceptKCP()
		if err != nil {
			if !s.running.Load() {
				break
			}
			continue
		}
		go s.serve(sess)
	}
	return nil
}

func (s *KcpServer) serve(conn net.Conn) {
	sess := conn.(*kcp.UDPSession)
	sess.SetStreamMode(true)
	sess.SetWriteDelay(false)
	sess.SetNoDelay(s.opts.NoDelay, s.opts.Interval, s.opts.Resend, s.opts.Nc)
	sess.SetACKNoDelay(s.opts.AckNoDelay)
	sess.SetMtu(s.opts.Mtu)
	sess.SetWindowSize(s.opts.SndWnd, s.opts.RevWnd)

	s.once.Do(func() {
		s.smuxConfig = smux.DefaultConfig()
		s.smuxConfig.Version = s.opts.SmuxVer
		s.smuxConfig.MaxReceiveBuffer = s.opts.SmuxBuf
		s.smuxConfig.MaxStreamBuffer = s.opts.StreamBuf
		s.smuxConfig.KeepAliveInterval = time.Duration(s.opts.KeepAlive) * time.Second
	})

	if !s.opts.NoCompress {
		conn = connection.NewCompressConn(conn)
	}

	// connection multiplexing
	muxSess, err := smux.Server(conn, s.smuxConfig)
	if err != nil {
		return
	}

	defer muxSess.Close()

	for {
		stream, err := muxSess.AcceptStream()
		if err != nil {
			return
		}
		cc := &mux.MuxStreamConn{
			Conn:   sess,
			Stream: stream,
		}

		go newConn(cc, s).serve()
	}
}

func (s *KcpServer) Serve(conn *Conn) {
	if s.Handler != nil {
		s.Handler.ServeKCP(conn)
	}
}
