package connection

import (
	"net"

	"github.com/quic-go/quic-go"
)

var _ net.Conn = (*QuicConn)(nil)

type QuicConn struct {
	quic.Stream
	laddr net.Addr
	raddr net.Addr
}

func NewQuicConn(stream quic.Stream, laddr, raddr net.Addr) *QuicConn {
	return &QuicConn{
		Stream: stream,
		laddr:  laddr,
		raddr:  raddr,
	}
}

func (c *QuicConn) LocalAddr() net.Addr { return c.laddr }

func (c *QuicConn) RemoteAddr() net.Addr { return c.raddr }
