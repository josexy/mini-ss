package transport

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/quic-go/quic-go"
)

type quicConn struct {
	addr string
	idx  int
	quic.EarlyConnection
}

type quicDialer struct {
	err       error
	tlsConfig *tls.Config
	cpool     *connPool[*quicConn]
	opts      *options.QuicOptions
}

func TlsConfigQuicALPN(config *tls.Config) *tls.Config {
	tlsConfig := config.Clone()
	tlsConfig.NextProtos = []string{"http/3", "quic/v1"}
	return tlsConfig
}

func newQUICDialer(opt options.Options) *quicDialer {
	opt.Update()
	quicOpts := opt.(*options.QuicOptions)
	tlsConfig, err := quicOpts.GetClientTlsConfig()
	if tlsConfig == nil {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(100)
	return &quicDialer{
		err:       err,
		opts:      quicOpts,
		tlsConfig: tlsConfig,
		cpool:     newConnPool[*quicConn](quicOpts.Conns),
	}
}

func (d *quicDialer) dial(ctx context.Context, addr string) (quic.EarlyConnection, error) {
	var raddr *net.UDPAddr
	var err error
	if raddr, err = resolver.DefaultResolver.ResolveUDPAddr(ctx, addr); err != nil {
		return nil, err
	}
	conn, err := ListenLocalUDP(ctx)
	if err != nil {
		return nil, err
	}

	return quic.DialEarly(ctx, conn, raddr, TlsConfigQuicALPN(d.tlsConfig), &quic.Config{
		HandshakeIdleTimeout:  d.opts.HandshakeIdleTimeout,
		KeepAlivePeriod:       d.opts.KeepAlivePeriod,
		MaxIdleTimeout:        d.opts.MaxIdleTimeout,
		MaxIncomingStreams:    1 << 32,
		MaxIncomingUniStreams: 1 << 32,
		Versions: []quic.Version{
			quic.Version1,
			quic.Version2,
		},
	})
}

func (d *quicDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	if d.err != nil {
		return nil, d.err
	}
	conn, err := d.getAndDial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return d.openStreamConn(ctx, conn)
}

func (d *quicDialer) getAndDial(ctx context.Context, addr string) (*quicConn, error) {
	return d.cpool.getConn(ctx, addr, func(ctx context.Context, addr string, idx int) (*quicConn, error) {
		c, err := d.dial(ctx, addr)
		if err != nil {
			return nil, err
		}
		return &quicConn{addr: addr, idx: idx, EarlyConnection: c}, nil
	})
}

func (d *quicDialer) retryDial(ctx context.Context, addr string, index int) (*quicConn, error) {
	return d.cpool.getConnWithIndex(ctx, addr, index, false, func(ctx context.Context, addr string, idx int) (*quicConn, error) {
		c, err := d.dial(ctx, addr)
		if err != nil {
			return nil, err
		}
		return &quicConn{addr: addr, idx: idx, EarlyConnection: c}, nil
	})
}

func (d *quicDialer) openStreamConn(ctx context.Context, conn *quicConn) (net.Conn, error) {
	var err error
	var stream quic.Stream
	var fails, retries = 0, 1
	for {
		newCtx, cancel := context.WithTimeout(ctx, time.Second*15)
		// Try to open quic connection stream
		if stream, err = conn.OpenStreamSync(newCtx); err == nil {
			cancel()
			break
		}
		cancel()

		if fails >= retries {
			return nil, err
		}
		// Reset the connection slot
		d.cpool.close(conn.idx, func(qc *quicConn) error {
			return conn.CloseWithError(quic.ApplicationErrorCode(0), err.Error())
		})
		// Reuse the same connection slot and retry connect to the server again
		newCtx, cancel = context.WithTimeout(ctx, time.Second*15)
		if conn, err = d.retryDial(newCtx, conn.addr, conn.idx); err != nil {
			cancel()
			return nil, err
		}
		cancel()
		fails++
	}
	logger.Logger.Tracef("quic open stream [%d] for conn: %s, idx:[%d]", stream.StreamID(), conn.LocalAddr(), conn.idx)
	return connection.NewQuicConn(stream, conn.LocalAddr(), conn.RemoteAddr()), nil
}
