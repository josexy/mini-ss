package transport

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/mux"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/resolver"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

type kcpDialer struct {
	sessions []*mux.MuxBindSession
	cfg      *smux.Config
	opts     *options.KcpOptions
	once     sync.Once
	rrIndex  uint16
	numConn  uint16
	reconn   int
}

func (d *kcpDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	d.once.Do(func() {
		d.opts.Update()

		d.cfg = smux.DefaultConfig()
		d.cfg.Version = d.opts.SmuxVer
		d.cfg.MaxReceiveBuffer = d.opts.SmuxBuf
		d.cfg.MaxStreamBuffer = d.opts.StreamBuf
		d.cfg.KeepAliveInterval = time.Duration(d.opts.KeepAlive) * time.Second

		d.sessions = make([]*mux.MuxBindSession, d.opts.Conns)
		d.rrIndex = 0
		d.numConn = uint16(d.opts.Conns)
		// retry connect count
		d.reconn = 3
	})

	idx := d.rrIndex % d.numConn

	if d.sessions[idx] != nil && d.sessions[idx].IsClose() {
		d.sessions[idx].Close()
	}

	// mux session uninitialized or closed
	if d.sessions[idx] == nil || d.sessions[idx].IsClose() {
		sess := d.waitForDial(ctx, addr)
		if sess == nil {
			return nil, fmt.Errorf("dial %s failed", addr)
		}
		d.sessions[idx] = sess
	}
	d.rrIndex++
	// open mux session connection
	return d.openStreamConn(d.sessions[idx])
}

func (d *kcpDialer) waitForDial(ctx context.Context, addr string) *mux.MuxBindSession {
	for i := 0; i < d.reconn; i++ {
		if sess, err := d.dial(ctx, addr); err == nil {
			return sess
		}
		time.Sleep(time.Second * 2)
	}
	return nil
}

func (d *kcpDialer) dialWithOptions(ctx context.Context, addr string) (*kcp.UDPSession, error) {
	conn, err := ListenLocalUDP(ctx)
	if err != nil {
		return nil, err
	}
	var raddr *net.UDPAddr
	if raddr, err = resolver.DefaultResolver.ResolveUDPAddr(ctx, addr); err != nil {
		return nil, err
	}
	return kcp.NewConn2(raddr, d.opts.BC, d.opts.DataShard, d.opts.ParityShard, conn)
}

func (d *kcpDialer) dial(ctx context.Context, addr string) (*mux.MuxBindSession, error) {
	conn, err := d.dialWithOptions(ctx, addr)
	if err != nil {
		return nil, err
	}
	conn.SetStreamMode(true)
	conn.SetWriteDelay(false)
	conn.SetNoDelay(d.opts.NoDelay, d.opts.Interval, d.opts.Resend, d.opts.Nc)
	conn.SetACKNoDelay(d.opts.AckNoDelay)
	conn.SetMtu(d.opts.Mtu)
	conn.SetWindowSize(d.opts.SndWnd, d.opts.RevWnd)
	conn.SetReadBuffer(d.opts.SockBuf)
	conn.SetWriteBuffer(d.opts.SockBuf)
	if d.opts.Dscp > 0 {
		conn.SetDSCP(d.opts.Dscp)
	}

	var cc net.Conn = conn
	if !d.opts.NoCompress {
		cc = connection.NewCompressConn(conn)
	}
	sess, err := smux.Client(cc, d.cfg)
	if err != nil {
		return nil, err
	}
	return mux.NewMuxBindSession(conn, sess), nil
}

func (d *kcpDialer) openStreamConn(sess *mux.MuxBindSession) (net.Conn, error) {
	cc, err := sess.GetStreamConn()
	if err != nil {
		sess.Close()
		return nil, err
	}
	return cc, nil
}
