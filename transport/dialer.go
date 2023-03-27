package transport

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/netstackgo/bind"
)

const (
	Tcp Type = iota // default
	Kcp
	Quic
	Websocket
	Obfs
)

const dialTimeout = 5 * time.Second

type Type uint8

func (t Type) String() string {
	switch t {
	case Tcp:
		return "tcp"
	case Kcp:
		return "kcp"
	case Quic:
		return "quic"
	case Websocket:
		return "websocket"
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
	case Tcp:
		d.d = &tcpDialer{}
	case Kcp:
		d.d = &kcpDialer{Opts: opt.(*KcpOptions)}
	case Websocket:
		d.d = &wsDialer{Opts: opt.(*WsOptions)}
	case Quic:
		d.d = &quicDialer{Opts: opt.(*QuicOptions)}
	case Obfs:
		d.d = &obfsDialer{Opts: opt.(*ObfsOptions)}
	}
	return d
}

func (d *xDialer) Dial(addr string) (net.Conn, error) { return d.d.Dial(addr) }

func resolveIP(host string) netip.Addr {
	var ip netip.Addr
	var err error
	// the host may be a ip address or domain name
	if ip, err = netip.ParseAddr(host); err != nil {
		ip = resolver.DefaultResolver.ResolveHost(host)
	}
	return ip
}

func resolveUDPAddr(addr string) (*net.UDPAddr, error) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ip := resolveIP(host)
	if !ip.IsValid() {
		return nil, fmt.Errorf("can not lookup host: %s", addr)
	}
	port, _ := strconv.ParseUint(p, 10, 16)
	return &net.UDPAddr{
		IP:   ip.AsSlice(),
		Port: int(port),
	}, nil
}

func DialTCP(addr string) (net.Conn, error) {
	d := tcpDialer{}
	return d.Dial(addr)
}

// ListenLocalUDP create an unconnected udp connection
func ListenLocalUDP() (net.PacketConn, error) {
	if DefaultDialerOutboundOption.Interface == "" {
		return net.ListenPacket("udp", "")
	}
	var lc net.ListenConfig
	addr, err := bind.BindToDeviceForUDP(DefaultDialerOutboundOption.Interface, &lc, "udp", "")
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
