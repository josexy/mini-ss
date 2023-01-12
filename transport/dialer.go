package transport

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/util"
)

const (
	Default Type = iota // default TCP/UDP
	KCP
	QUIC
	Websocket
	Obfs
)

const dialTimeout = 5 * time.Second

// Type transport type between ss-local and ss-server
// supports TCP/KCP/QUIC/WS protocol
// for UDP relay, only the default transport type is supported
type Type byte

func (t Type) String() string {
	switch t {
	case Default:
		return "tcp"
	case KCP:
		return "kcp"
	case QUIC:
		return "quic"
	case Websocket:
		return "ws"
	case Obfs:
		return "obfs"
	default:
		return "unknown"
	}
}

type Dialer interface {
	Dial(addr string) (net.Conn, error)
}

type xDialer struct{ d Dialer }

func NewDialer(tr Type, opt Options) Dialer {
	d := new(xDialer)
	switch tr {
	case Default:
		d.d = &tcpDialer{}
	case KCP:
		d.d = &kcpDialer{Opts: opt.(*KcpOptions)}
	case Websocket:
		d.d = &wsDialer{Opts: opt.(*WsOptions)}
	case QUIC:
		d.d = &quicDialer{Opts: opt.(*QuicOptions)}
	case Obfs:
		d.d = &obfsDialer{Opts: opt.(*ObfsOptions)}
	}
	return d
}

func (d *xDialer) Dial(addr string) (net.Conn, error) {
	return d.d.Dial(addr)
}

func resolveIP(host string) netip.Addr {
	var ip netip.Addr
	var err error
	// the host is ip address or domain name
	if ip, err = netip.ParseAddr(host); err != nil {
		ip = resolver.DefaultResolver.ResolveHost(host)
	}
	return ip
}

func resolveUDPAddr(addr string) (*net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ip := resolveIP(host)
	if !ip.IsValid() {
		return nil, fmt.Errorf("can not lookup host: %s", addr)
	}
	return &net.UDPAddr{
		IP:   ip.AsSlice(),
		Port: util.MustStringToInt(port),
	}, nil
}

func DialTCP(addr string) (net.Conn, error) {
	d := tcpDialer{}
	return d.Dial(addr)
}

// DialLocalUDP create an unconnected udp connection
func DialLocalUDP() (net.PacketConn, error) {
	if DefaultDialerOutboundOption.Interface == "" {
		return net.ListenPacket("udp", "")
	}
	var lc net.ListenConfig
	// bind outbound interface to a packet socket
	addr, err := bindIfaceToListenConfig(DefaultDialerOutboundOption.Interface, &lc, "udp", "")
	if err != nil {
		return nil, err
	}
	return lc.ListenPacket(context.Background(), "udp", addr)
}

// DialUDP create a connected udp connection
func DialUDP(address string) (*net.UDPConn, error) {
	rAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}
	con, err := net.DialUDP("udp", nil, rAddr)
	if err != nil {
		return nil, err
	}
	return con, nil
}
