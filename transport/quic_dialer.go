package transport

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/quic-go/quic-go"
)

type quicConn struct {
	addr string
	quic.EarlyConnection
}

type quicDialer struct {
	once      sync.Once
	rrIndex   uint16
	numConn   uint16
	tlsConfig *tls.Config
	tlsErr    error
	conns     []*quicConn
	opts      *QuicOptions
}

func TlsConfigQuicALPN(config *tls.Config) *tls.Config {
	tlsConfig := config.Clone()
	tlsConfig.NextProtos = []string{"http/3", "quic/v1"}
	return tlsConfig
}

func (d *quicDialer) dial(ctx context.Context, addr string) (quic.EarlyConnection, error) {
	var raddr *net.UDPAddr
	var err error
	if raddr, err = resolveUDPAddr(addr); err != nil {
		return nil, err
	}
	conn, err := ListenLocalUDP(ctx)
	if err != nil {
		return nil, err
	}

	return quic.DialEarly(ctx, conn, raddr, TlsConfigQuicALPN(d.tlsConfig), &quic.Config{
		HandshakeIdleTimeout: d.opts.HandshakeIdleTimeout,
		KeepAlivePeriod:      d.opts.KeepAlivePeriod,
		MaxIdleTimeout:       d.opts.MaxIdleTimeout,
		Versions: []quic.Version{
			quic.Version1,
			quic.Version2,
		},
	})
}

func (d *quicDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	d.once.Do(func() {
		d.opts.Update()
		d.rrIndex = 0
		d.numConn = uint16(d.opts.Conns)
		d.conns = make([]*quicConn, 32)
		d.tlsConfig, d.tlsErr = d.opts.GetClientTlsConfig()
		if d.tlsErr != nil {
			return
		}
		if d.tlsConfig == nil {
			d.tlsConfig = &tls.Config{InsecureSkipVerify: true}
		}
	})
	if d.tlsErr != nil {
		return nil, d.tlsErr
	}

	idx := d.rrIndex % d.numConn
	if _, err := d.dialAndStore(ctx, addr, idx); err != nil {
		return nil, err
	}
	d.rrIndex++

	return d.openStreamConn(ctx, idx)
}

func (d *quicDialer) dialAndStore(ctx context.Context, addr string, index uint16) (*quicConn, error) {
	if d.conns[index] == nil {
		conn, err := d.dial(ctx, addr)
		if err != nil {
			logger.Logger.ErrorBy(err)
			return nil, err
		}
		logger.Logger.Tracef("quic dial new conn: %s", conn.LocalAddr())
		d.conns[index] = &quicConn{addr: addr, EarlyConnection: conn}
	}
	return d.conns[index], nil
}

func (d *quicDialer) openStreamConn(ctx context.Context, index uint16) (net.Conn, error) {
	conn := d.conns[index]
	var err error
	var stream quic.Stream
	var fails, retries = 0, 1
	for {
		// Try to open quic connection stream
		if stream, err = conn.OpenStreamSync(ctx); err == nil {
			break
		}
		if fails >= retries {
			return nil, errors.New("failed to open quic stream")
		}
		_ = conn.CloseWithError(quic.ApplicationErrorCode(0), "")
		// Reset the connection slot
		d.conns[index] = nil
		// Reuse the same connection slot and retry connect to the server again
		if conn, err = d.dialAndStore(ctx, conn.addr, index); err != nil {
			return nil, err
		}
		fails++
	}
	logger.Logger.Tracef("quic open stream [%d] for conn: %s", stream.StreamID(), conn.LocalAddr())
	return connection.NewQuicConn(stream, conn.LocalAddr(), conn.RemoteAddr()), nil
}
