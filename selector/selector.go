package selector

import (
	"net"

	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/mini-ss/util/ordmap"
)

var ProxySelector = NewSelector()

type (
	SelectorHandler       func(net.Conn, string) error
	SelectorPacketHandler func(net.PacketConn, string) error
)

type Selector struct {
	tcpDirector      *relay.TCPDirectRelayer
	udpDirector      *relay.UDPDirectRelayer
	tcpDirectHandler SelectorHandler
	udpDirectHandler SelectorPacketHandler
	tcpProxyNode     ordmap.OrderedMap
	udpProxyNode     ordmap.OrderedMap
}

func NewSelector() *Selector {
	selector := &Selector{
		tcpDirector: relay.NewTCPDirectRelayer(),
		udpDirector: relay.NewUDPDirectRelayer(),
	}
	selector.tcpDirectHandler = func(relayer net.Conn, remoteAddr string) error {
		return selector.tcpDirector.RelayDirectTCP(relayer, remoteAddr)
	}
	selector.udpDirectHandler = func(relayer net.PacketConn, remoteAddr string) error {
		return selector.udpDirector.RelayDirectUDP(relayer, remoteAddr)
	}
	return selector
}

func (selector *Selector) AddProxy(name string, ctx ctxv.V) {
	selector.tcpProxyNode.Store(name, relay.DstTCPRelayer{
		DstAddr: ctx.Addr,
		TCPRelayer: relay.NewTCPRelayer(
			ctx.Type,
			ctx.Options,
			nil,
			ctx.TcpConnBound,
		),
	})
}

func (selector *Selector) AddPacketProxy(name string, ctx ctxv.V) {
	selector.udpProxyNode.Store(name, relay.DstUDPRelayer{
		DstAddr:    ctx.Addr,
		UDPRelayer: relay.NewUDPRelayer(ctx.UdpConnBound),
	})
}

func (selector *Selector) Select(proxy string) SelectorHandler {
	if proxy == "" {
		return selector.tcpDirectHandler
	}
	value, ok := selector.tcpProxyNode.Load(proxy)
	if !ok {
		logger.Logger.Warnf("tcp: try to connect directly since proxy %q not found", proxy)
		return selector.tcpDirectHandler
	}
	tcpRelayer := value.(relay.DstTCPRelayer)
	return func(relayer net.Conn, remoteAddr string) error {
		return tcpRelayer.RelayLocalToServer(
			relayer,
			tcpRelayer.DstAddr,
			remoteAddr,
		)
	}
}

func (selector *Selector) SelectPacket(proxy string) SelectorPacketHandler {
	if proxy == "" {
		return selector.udpDirectHandler
	}
	value, ok := selector.udpProxyNode.Load(proxy)
	if !ok {
		logger.Logger.Warnf("udp: try to connect directly since proxy %q not found or udp relay disabled", proxy)
		return selector.udpDirectHandler
	}
	udpRelayer := value.(relay.DstUDPRelayer)
	return func(relayer net.PacketConn, remoteAddr string) error {
		return udpRelayer.RelayLocalToServer(
			relayer,
			udpRelayer.DstAddr,
			remoteAddr,
		)
	}
}
