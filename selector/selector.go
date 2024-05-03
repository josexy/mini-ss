package selector

import (
	"net"

	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/mini-ss/util/ordmap"
)

var ProxySelector = NewSelector()

type StreamInvoker interface {
	Invoke(net.Conn, string) error
}
type PacketInvoker interface {
	Invoke(net.PacketConn, string) error
}

type StreamInvokerFunc func(net.Conn, string) error
type PacketInvokerFunc func(net.PacketConn, string) error

func (f StreamInvokerFunc) Invoke(c net.Conn, s string) error       { return f(c, s) }
func (f PacketInvokerFunc) Invoke(c net.PacketConn, s string) error { return f(c, s) }

type Selector struct {
	tcpDirector  *relay.TCPDirectRelayer
	udpDirector  *relay.UDPDirectRelayer
	tcpProxyNode ordmap.OrderedMap
	udpProxyNode ordmap.OrderedMap
}

func NewSelector() *Selector {
	selector := &Selector{
		tcpDirector: relay.NewTCPDirectRelayer(),
		udpDirector: relay.NewUDPDirectRelayer(),
	}
	return selector
}

func (selector *Selector) AddProxy(proxy string, ctx ctxv.V) {
	selector.tcpProxyNode.Store(proxy, relay.NewProxyTCPRelayer(
		ctx.Addr,
		ctx.Type,
		ctx.Options,
		nil,
		ctx.TcpConnBound,
	))
}

func (selector *Selector) AddPacketProxy(proxy string, ctx ctxv.V) {
	selector.udpProxyNode.Store(proxy, relay.NewProxyUDPRelayer(
		ctx.Addr,
		nil,
		ctx.UdpConnBound,
	))
}

func (selector *Selector) Select(proxy string) StreamInvoker {
	if proxy == "" {
		logger.Logger.Trace("tcp: direct")
		return StreamInvokerFunc(selector.tcpDirector.RelayToServer)
	}
	node, ok := selector.tcpProxyNode.Load(proxy)
	if !ok {
		logger.Logger.Trace("tcp: direct")
		logger.Logger.Warnf("tcp: try to connect directly since proxy %q not found", proxy)
		return StreamInvokerFunc(selector.tcpDirector.RelayToServer)
	}
	logger.Logger.Trace("tcp: proxy")
	return StreamInvokerFunc(node.(*relay.ProxyTCPRelayer).RelayToProxyServer)
}

func (selector *Selector) SelectPacket(proxy string) PacketInvoker {
	if proxy == "" {
		logger.Logger.Trace("udp: direct")
		return PacketInvokerFunc(selector.udpDirector.RelayToServer)
	}
	node, ok := selector.udpProxyNode.Load(proxy)
	if !ok {
		logger.Logger.Trace("udp: direct")
		logger.Logger.Warnf("udp: try to connect directly since proxy %q not found or udp relay disabled", proxy)
		return PacketInvokerFunc(selector.udpDirector.RelayToServer)
	}
	logger.Logger.Trace("udp: proxy")
	return PacketInvokerFunc(node.(*relay.ProxyUDPRelayer).RelayToProxyServer)
}
