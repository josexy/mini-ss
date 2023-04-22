package transport

import (
	"context"
	"crypto/tls"
	"net"
	"sync"

	"github.com/josexy/mini-ss/connection"
	"github.com/quic-go/quic-go"
)

type quicDialer struct {
	sessions []quic.EarlyConnection
	once     sync.Once
	rrIndex  uint16
	numConn  uint16
	Opts     *QuicOptions
}

func TlsConfigQuicALPN(config *tls.Config) *tls.Config {
	tlsConfig := config.Clone()
	tlsConfig.NextProtos = []string{"http/3", "quic/v1"}
	return tlsConfig
}

func (d *quicDialer) dial(addr string) (quic.EarlyConnection, error) {
	var raddr *net.UDPAddr
	var err error
	if raddr, err = resolveUDPAddr(addr); err != nil {
		return nil, err
	}
	quicConfig := &quic.Config{
		HandshakeIdleTimeout: d.Opts.HandshakeIdleTimeout,
		KeepAlivePeriod:      d.Opts.KeepAlivePeriod,
		MaxIdleTimeout:       d.Opts.MaxIdleTimeout,
		Versions: []quic.VersionNumber{
			quic.Version1,
			quic.VersionDraft29,
		},
	}
	var tlsConfig *tls.Config
	if tlsConfig == nil {
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
	}
	conn, err := ListenLocalUDP()
	if err != nil {
		return nil, err
	}

	return quic.DialEarly(conn, raddr, raddr.String(), TlsConfigQuicALPN(tlsConfig), quicConfig)
}

func (d *quicDialer) Dial(addr string) (net.Conn, error) {

	d.once.Do(func() {
		d.Opts.Update()
		d.rrIndex = 0
		d.numConn = uint16(d.Opts.Conns)
		d.sessions = make([]quic.EarlyConnection, d.numConn)
	})

	idx := d.rrIndex % d.numConn
	if d.sessions[idx] == nil {
		session, err := d.dial(addr)
		if err != nil {
			return nil, err
		}
		d.sessions[idx] = session
	}
	d.rrIndex++

	return d.openStreamConn(d.sessions[idx])
}

func (d *quicDialer) openStreamConn(session quic.Connection) (net.Conn, error) {
	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return nil, err
	}
	return connection.NewQuicConn(
		stream,
		session.LocalAddr(),
		session.RemoteAddr(),
	), nil
}
