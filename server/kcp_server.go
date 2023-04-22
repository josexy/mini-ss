package server

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/mux"
	"github.com/josexy/mini-ss/transport"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

type KcpServer struct {
	net.Listener
	Addr       string
	Handler    KcpHandler
	mu         sync.Mutex
	closed     uint32
	doneChan   chan struct{}
	smuxConfig *smux.Config
	once       sync.Once
	opts       *transport.KcpOptions
	err        chan error
}

func NewKcpServer(addr string, handler KcpHandler, opts transport.Options) *KcpServer {
	return &KcpServer{
		Addr:     addr,
		Handler:  handler,
		doneChan: make(chan struct{}),
		err:      make(chan error, 1),
		opts:     opts.((*transport.KcpOptions)),
		closed:   1,
	}
}

func (s *KcpServer) LocalAddr() string { return s.Addr }

func (s *KcpServer) Type() ServerType { return Kcp }

func (s *KcpServer) Close() error {
	if atomic.LoadUint32(&s.closed) != 0 {
		return ErrServerClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	atomic.StoreUint32(&s.closed, 1)
	close(s.doneChan)

	return s.Listener.Close()
}

func (s *KcpServer) Error() chan error { return s.err }

func (s *KcpServer) Build() Server { return s }

func (s *KcpServer) Start() {
	s.opts.Update()

	ln, err := kcp.ListenWithOptions(s.Addr, s.opts.BC, s.opts.DataShard, s.opts.ParityShard)
	if err != nil {
		s.err <- err
		return
	}
	s.Listener = ln

	if s.opts.Dscp > 0 {
		ln.SetDSCP(s.opts.Dscp)
	}
	ln.SetReadBuffer(s.opts.SockBuf)
	ln.SetWriteBuffer(s.opts.SockBuf)

	s.err <- nil
	atomic.StoreUint32(&s.closed, 0)
	defer s.Close()
	for {
		sess, err := ln.AcceptKCP()
		if err != nil {
			select {
			case <-s.getDoneChan():
				return
			default:
			}
			continue
		}
		go s.serve(sess)
	}
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

		conn := newConn(cc, s)
		go conn.serve()
	}
}

func (s *KcpServer) Serve(c *Conn) {
	if s.Handler != nil {
		s.Handler.ServeKCP(c.conn)
	}
}

func (s *KcpServer) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}
