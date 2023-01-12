package protocol

import (
	"bytes"
	"net"
)

type PacketConn struct {
	net.PacketConn
	Protocol
}

func (c *PacketConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	buf := &bytes.Buffer{}

	err := c.EncodePacket(buf, b)
	if err != nil {
		return 0, err
	}
	_, err = c.PacketConn.WriteTo(buf.Bytes(), addr)
	return len(b), err
}

func (c *PacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	decoded, err := c.DecodePacket(b[:n])
	if err != nil {
		return n, addr, err
	}
	copy(b, decoded)
	return len(decoded), addr, nil
}
