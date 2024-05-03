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
	Grpc
)

const DefaultDialTimeout = 10 * time.Second

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
	case Grpc:
		return "grpc"
	default:
		return "unknown"
	}
}

type Dialer interface {
	Dial(context.Context, string) (net.Conn, error)
}

func NewDialer(tr Type, opt Options) Dialer {
	var dialer Dialer
	switch tr {
	case Tcp:
		dialer = &tcpDialer{}
	case Kcp:
		dialer = &kcpDialer{opts: opt.(*KcpOptions)}
	case Websocket:
		dialer = &wsDialer{opts: opt.(*WsOptions)}
	case Quic:
		dialer = &quicDialer{opts: opt.(*QuicOptions)}
	case Obfs:
		dialer = &obfsDialer{opts: opt.(*ObfsOptions)}
	case Grpc:
		dialer = &grpcDialer{opts: opt.(*GrpcOptions)}
	default:
		panic("not supported type")
	}
	return dialer
}

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

func DialTCP(ctx context.Context, addr string) (net.Conn, error) {
	d := tcpDialer{}
	return d.Dial(ctx, addr)
}

// ListenUDP create an unconnected udp connection with the specified local addr
func ListenUDP(ctx context.Context, addr string) (net.PacketConn, error) {
	if DefaultDialerOutboundOption.Interface == "" {
		return (&net.ListenConfig{}).ListenPacket(ctx, "udp", addr)
	}
	var lc net.ListenConfig
	addr, err := bind.BindToDeviceForUDP(DefaultDialerOutboundOption.Interface, &lc, "udp", addr)
	if err != nil {
		return nil, err
	}
	return lc.ListenPacket(ctx, "udp", addr)
}

// ListenLocalUDP create an unconnected udp connection with random local addr
func ListenLocalUDP(ctx context.Context) (net.PacketConn, error) {
	return ListenUDP(ctx, "")
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
