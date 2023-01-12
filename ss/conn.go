package ss

import (
	"net"

	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/ss/aead"
	"github.com/josexy/mini-ss/ss/stream"
	"github.com/josexy/mini-ss/ssr"
	"github.com/josexy/mini-ss/transport"
)

type defaultConn struct{}

func (*defaultConn) TcpConn(c net.Conn) net.Conn { return c }

func (*defaultConn) UdpConn(c net.PacketConn) net.PacketConn { return c }

func makeStreamConn(scipher cipher.StreamCipher, acipher cipher.AEADCipher) transport.TcpConnBound {
	return transport.TcpConnBoundHandler(func(c net.Conn) net.Conn {
		if scipher != nil {
			return stream.NewStreamConn(c, scipher)
		} else if acipher != nil {
			return aead.NewStreamConn(c, acipher)
		}
		return new(defaultConn).TcpConn(c)
	})
}

func makePacketConn(scipher cipher.StreamCipher, acipher cipher.AEADCipher) transport.UdpConnBound {
	return transport.UdpConnBoundHandler(func(c net.PacketConn) net.PacketConn {
		if scipher != nil {
			return stream.NewPacketConn(c, scipher)
		} else if acipher != nil {
			return aead.NewPacketConn(c, acipher)
		}
		return new(defaultConn).UdpConn(c)
	})
}

func makeSSRClientStreamConn(scipher *ssr.SSRClientStreamCipher) transport.TcpConnBound {
	return transport.TcpConnBoundHandler(func(c net.Conn) net.Conn {
		ssr := &ssr.ShadowsocksR{
			SSTcp:    makeStreamConn(scipher.StreamCipher, nil),
			Cipher:   scipher,
			Obfs:     scipher.Obfs,
			Protocol: scipher.Proto,
		}
		return ssr.StreamConn(c)
	})
}

func makeSSRClientPacketConn(scipher *ssr.SSRClientStreamCipher) transport.UdpConnBound {
	return transport.UdpConnBoundHandler(func(c net.PacketConn) net.PacketConn {
		ssr := &ssr.ShadowsocksR{
			SSUdp:    makePacketConn(scipher.StreamCipher, nil),
			Cipher:   scipher,
			Protocol: scipher.Proto,
		}
		return ssr.PacketConn(c)
	})
}
