package server

import (
	"errors"
	"net"

	"github.com/josexy/mini-ss/bufferpool"
)

type ServerType byte

const (
	Tcp ServerType = iota
	Udp
	Socks
	Http
	SimpleTcpTun
	SimpleUdpTun
	Kcp
	Ws
	Obfs
	Quic
	Mixed
)

func (t ServerType) String() string {
	switch t {
	case Tcp:
		return "tcp"
	case Udp:
		return "udp"
	case Socks:
		return "socks"
	case Http:
		return "http"
	case SimpleTcpTun:
		return "simple-tcp-tun"
	case SimpleUdpTun:
		return "simple-udp-tun"
	case Kcp:
		return "kcp"
	case Ws:
		return "ws"
	case Obfs:
		return "obfs"
	case Quic:
		return "quic"
	case Mixed:
		return "mixed-socks-http"
	}
	return "unknown"
}

var (
	stackTraceBufferPool = bufferpool.NewBufferPool(4096)
	ErrServerClosed      = errors.New("server: Server closed")
)

type (
	Server interface {
		Build() Server
		Start()
		Error() chan error
		Close() error
		Serve(*Conn)
		LocalAddr() string
		Type() ServerType
	}
	TcpHandler      interface{ ServeTCP(net.Conn) }
	UdpHandler      interface{ ServeUDP(net.PacketConn) }
	KcpHandler      interface{ ServeKCP(net.Conn) }
	WsHandler       interface{ ServeWS(net.Conn) }
	ObfsHandler     interface{ ServeOBFS(net.Conn) }
	QuicHandler     interface{ ServeQUIC(net.Conn) }
	TcpHandlerFunc  func(net.Conn)
	UdpHandlerFunc  func(net.PacketConn)
	KcpHandlerFunc  func(net.Conn)
	WsHandlerFunc   func(net.Conn)
	ObfsHandlerFunc func(net.Conn)
	QuicHandlerFunc func(net.Conn)
)

func (f TcpHandlerFunc) ServeTCP(conn net.Conn)       { f(conn) }
func (f UdpHandlerFunc) ServeUDP(conn net.PacketConn) { f(conn) }
func (f KcpHandlerFunc) ServeKCP(conn net.Conn)       { f(conn) }
func (f WsHandlerFunc) ServeWS(conn net.Conn)         { f(conn) }
func (f ObfsHandlerFunc) ServeOBFS(conn net.Conn)     { f(conn) }
func (f QuicHandlerFunc) ServeQUIC(conn net.Conn)     { f(conn) }
