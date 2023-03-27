package transport

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/josexy/mini-ss/mux"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
)

type kcpDialer struct {
	sessions []*mux.MuxBindSession
	cfg      *smux.Config
	Opts     *KcpOptions
	once     sync.Once
	rrIndex  uint16
	numConn  uint16
	reconn   int
}

func (d *kcpDialer) Dial(addr string) (net.Conn, error) {
	d.once.Do(func() {
		d.Opts.Update()

		d.cfg = smux.DefaultConfig()
		d.cfg.Version = d.Opts.SmuxVer
		d.cfg.MaxReceiveBuffer = d.Opts.SmuxBuf
		d.cfg.MaxStreamBuffer = d.Opts.StreamBuf
		d.cfg.KeepAliveInterval = time.Duration(d.Opts.KeepAlive) * time.Second

		d.sessions = make([]*mux.MuxBindSession, d.Opts.Conns)
		d.rrIndex = 0
		d.numConn = uint16(d.Opts.Conns)
		// retry connect count
		d.reconn = 3
	})

	idx := d.rrIndex % d.numConn

	if d.sessions[idx] != nil && d.sessions[idx].IsClose() {
		d.sessions[idx].Close()
	}

	// mux session uninitialized or closed
	if d.sessions[idx] == nil || d.sessions[idx].IsClose() {
		sess := d.waitForDial(addr)
		if sess == nil {
			return nil, fmt.Errorf("dial %s failed", addr)
		}
		d.sessions[idx] = sess
	}
	d.rrIndex++
	// open mux session connection
	return d.openStreamConn(d.sessions[idx])
}

func (d *kcpDialer) waitForDial(addr string) *mux.MuxBindSession {
	for i := 0; i < d.reconn; i++ {
		if sess, err := d.dial(addr); err == nil {
			return sess
		}
		time.Sleep(time.Second * 2)
	}
	return nil
}

func (d *kcpDialer) dialWithOptions(addr string) (*kcp.UDPSession, error) {
	conn, err := ListenLocalUDP()
	if err != nil {
		return nil, err
	}
	var raddr *net.UDPAddr
	if raddr, err = resolveUDPAddr(addr); err != nil {
		return nil, err
	}
	return kcp.NewConn2(raddr, d.Opts.BC, d.Opts.DataShard, d.Opts.ParityShard, conn)
}

func (d *kcpDialer) dial(addr string) (*mux.MuxBindSession, error) {
	conn, err := d.dialWithOptions(addr)
	if err != nil {
		return nil, err
	}
	conn.SetStreamMode(true)
	conn.SetWriteDelay(false)
	conn.SetNoDelay(d.Opts.NoDelay, d.Opts.Interval, d.Opts.Resend, d.Opts.Nc)
	conn.SetACKNoDelay(d.Opts.AckNoDelay)
	conn.SetMtu(d.Opts.Mtu)
	conn.SetWindowSize(d.Opts.SndWnd, d.Opts.RevWnd)
	conn.SetReadBuffer(d.Opts.SockBuf)
	conn.SetWriteBuffer(d.Opts.SockBuf)
	if d.Opts.Dscp > 0 {
		conn.SetDSCP(d.Opts.Dscp)
	}

	var cc net.Conn = conn
	if !d.Opts.NoCompress {
		cc = NewCompressConn(conn)
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
