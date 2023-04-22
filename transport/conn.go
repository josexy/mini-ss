package transport

import (
	"net"
)

type Conn interface {
	net.Conn
	TCP() *net.TCPConn
	UDP() *net.UDPConn
}

type TcpConnBound interface {
	TcpConn(net.Conn) net.Conn
}

type UdpConnBound interface {
	UdpConn(net.PacketConn) net.PacketConn
}

type TcpConnBoundHandler func(net.Conn) net.Conn

func (f TcpConnBoundHandler) TcpConn(c net.Conn) net.Conn { return f(c) }

type UdpConnBoundHandler func(net.PacketConn) net.PacketConn

func (f UdpConnBoundHandler) UdpConn(c net.PacketConn) net.PacketConn { return f(c) }
