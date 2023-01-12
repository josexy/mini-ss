package ssr

import (
	"net"

	"github.com/josexy/mini-ss/ss/stream"
	"github.com/josexy/mini-ss/ssr/obfs"
	"github.com/josexy/mini-ss/ssr/protocol"
	"github.com/josexy/mini-ss/transport"
)

type ShadowsocksR struct {
	Cipher   *SSRClientStreamCipher
	SSTcp    transport.TcpConnBound
	SSUdp    transport.UdpConnBound
	Obfs     obfs.Obfs
	Protocol protocol.Protocol
}

func (ssr *ShadowsocksR) StreamConn(c net.Conn) net.Conn {
	c = ssr.Obfs.StreamConn(c)
	c = ssr.SSTcp.TcpConn(c)

	var iv []byte
	var err error

	// ssr only support stream cipher, can not aead cipher
	if streamConn, ok := c.(*stream.StreamConn); ok {
		iv, err = streamConn.ObtainWriteIV()
	}

	if err != nil {
		return nil
	}

	c = ssr.Protocol.StreamConn(c, iv)
	return c
}

// PacketConn the UDP relay in shadowsocksr only support stream cipher
// and can not use Obfs
func (ssr *ShadowsocksR) PacketConn(c net.PacketConn) net.PacketConn {
	c = ssr.SSUdp.UdpConn(c)
	c = ssr.Protocol.PacketConn(c)
	return c
}
