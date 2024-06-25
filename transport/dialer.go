package transport

import (
	"context"
	"net"
	"time"

	"github.com/josexy/cropstun/bind"
	"github.com/josexy/mini-ss/options"
)

const (
	Tcp Type = iota // default
	Quic
	Websocket
	Obfs
	Grpc
	Ssh
)

const DefaultDialTimeout = 10 * time.Second

type Type uint8

func (t Type) String() string {
	switch t {
	case Tcp:
		return "tcp"
	case Quic:
		return "quic"
	case Websocket:
		return "websocket"
	case Obfs:
		return "obfs"
	case Grpc:
		return "grpc"
	case Ssh:
		return "ssh"
	default:
		return "unknown"
	}
}

type Dialer interface {
	Dial(context.Context, string) (net.Conn, error)
}

func NewDialer(tr Type, opt options.Options) Dialer {
	var dialer Dialer
	switch tr {
	case Tcp:
		dialer = new(tcpDialer)
	case Websocket:
		dialer = newWSDialer(opt)
	case Quic:
		dialer = newQUICDialer(opt)
	case Grpc:
		dialer = newGRPCDialer(opt)
	case Ssh:
		dialer = newSSHDialer(opt)
	case Obfs:
		dialer = newOBFSDialer(opt)
	default:
		panic("unsupported transport dialer type")
	}
	return dialer
}

func DialTCP(ctx context.Context, addr string) (net.Conn, error) {
	d := tcpDialer{}
	return d.Dial(ctx, addr)
}

// ListenUDP create an unconnected udp connection with the specified local addr
func ListenUDP(ctx context.Context, addr string) (net.PacketConn, error) {
	if options.DefaultOptions.OutboundInterface == "" {
		return (&net.ListenConfig{}).ListenPacket(ctx, "udp", addr)
	}
	var lc net.ListenConfig
	addr, err := bind.BindToDeviceForPacket(options.DefaultOptions.OutboundInterface, &lc, "udp", addr)
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
