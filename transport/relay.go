package transport

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/socks/constant"
)

var largePool = bufferpool.NewBufferPool(16 * 1024)

func RelayTCP(dst, src io.ReadWriteCloser) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	fn := func(dest, src io.ReadWriteCloser) {
		defer wg.Done()
		_ = ioCopyWithBuffer(dest, src)
		_ = dest.Close()
	}
	go fn(dst, src)
	go fn(src, dst)
	wg.Wait()
}

func ioCopyWithBuffer(dst io.Writer, src io.Reader) error {
	buf := largePool.Get()
	defer largePool.Put(buf)
	_, err := io.CopyBuffer(dst, src, *buf)
	return err
}

type DstAddrRelayer struct {
	DstAddr string
	*TCPRelayer
	*UDPRelayer
	*TCPDirectRelayer
	*UDPDirectRelayer
}

type TCPDirectRelayer struct{ Dialer }

func NewTCPDirectRelayer() *TCPDirectRelayer {
	return &TCPDirectRelayer{Dialer: NewDialer(Default, nil)}
}

func (r *TCPDirectRelayer) RelayDirectTCP(relayer net.Conn, targetAddr string) error {
	dstConn, err := r.Dial(targetAddr)
	if err != nil {
		return err
	}
	logx.Info("[%s] [%s] <-> [%s]",
		color.BlueString("tcp-direct"),
		color.GreenString(relayer.RemoteAddr().String()),
		color.YellowString(targetAddr),
	)
	RelayTCP(dstConn, relayer)
	return nil
}

type TCPRelayer struct {
	Dialer
	relayType int
	transport Type
	inbound   TcpConnBound
	outbound  TcpConnBound
}

func NewTCPRelayer(relayType int, tr Type, opts Options, inbound, outbound TcpConnBound) *TCPRelayer {
	return &TCPRelayer{
		Dialer:    NewDialer(tr, opts),
		relayType: relayType,
		transport: tr,
		inbound:   inbound,
		outbound:  outbound,
	}
}

func (r *TCPRelayer) RelayTCP(relayer net.Conn, targetAddr, remoteAddr string) (err error) {
	if r.inbound != nil {
		relayer = r.inbound.TcpConn(relayer)
	}

	var dstConn net.Conn
	var remoteAddrForDebug = remoteAddr

	switch r.relayType {
	case constant.TCPSSLocalToSSServer:
		dstConn, err = r.Dial(targetAddr)
		if err != nil {
			return
		}
		if r.outbound != nil {
			dstConn = r.outbound.TcpConn(dstConn)
		}

		addr := address.ParseAddress1(remoteAddr)
		dstConn.Write(addr)

		// for debugging
		remoteAddr = targetAddr
		targetAddr = relayer.LocalAddr().String()

	case constant.TCPSSServerToTCPServer:
		addr, buf, er := address.ParseAddress4(relayer)
		if er != nil {
			return er
		}
		remoteAddr = addr.String()
		address.PutAddrBuf(buf)

		dstConn, err = r.Dial(remoteAddr)
		if err != nil {
			return
		}
		if r.outbound != nil {
			dstConn = r.outbound.TcpConn(dstConn)
		}
	}

	logx.Info("[%s] [%s] <-> [%s] <-> [%s] <-> [%s]",
		color.BlueString(r.transport.String()),
		color.GreenString(relayer.RemoteAddr().String()),
		color.YellowString(targetAddr),
		color.RedString(remoteAddr),
		color.CyanString(remoteAddrForDebug),
	)
	RelayTCP(dstConn, relayer)
	return nil
}

type UDPDirectRelayer struct{ natM *UdpDirectNATMap }

func NewUDPDirectRelayer() *UDPDirectRelayer {
	return &UDPDirectRelayer{natM: NewUdpDirectNATMap(time.Second * 10)}
}

func (r *UDPDirectRelayer) RelayDirectUDP(relayer net.PacketConn, targetAddr string) error {

	buf := r.natM.BufferPool.Get()
	defer r.natM.BufferPool.Put(buf)

	n, srcAddr, err := relayer.ReadFrom(*buf)
	if err != nil {
		return err
	}
	var index int
	if targetAddr == "" {
		dstAddr := address.ParseAddress3((*buf)[3:n])
		targetAddr = dstAddr.String()
		index = 3 + len(dstAddr)
	}
	dst := r.natM.Get(srcAddr.String())
	if dst == nil {
		dst, err = DialLocalUDP()
		if err != nil {
			return err
		}

		// symmetric nat
		dst = newSymmetricNATPacketConn(dst, targetAddr)

		r.natM.Add(srcAddr, relayer, dst)

		logx.Info("[%s] [%s] <-> [%s] <-> [%s]",
			color.BlueString("udp-direct"),
			color.GreenString(srcAddr.String()),
			color.YellowString(relayer.LocalAddr().String()),
			color.RedString(targetAddr),
		)
	}
	daddr, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return err
	}
	dst.WriteTo((*buf)[index:n], daddr)
	return nil
}

type UDPRelayer struct {
	natM      *UdpNATMap
	relayType int
	transport Type
	inbound   UdpConnBound
	outbound  UdpConnBound
}

func NewUDPRelayer(relayType int, tr Type, inbound, outbound UdpConnBound) *UDPRelayer {
	if tr != Default {
		logx.Fatal("udp relay only support default transport type")
	}
	if relayType < constant.UDPSSLocalToSSServer || relayType > constant.UDPTunServerToSSServer {
		logx.Fatal("udp relay type invalid")
	}
	return &UDPRelayer{
		natM:      NewUdpNATMap(time.Second * 10),
		relayType: relayType,
		transport: tr,
		inbound:   inbound,
		outbound:  outbound,
	}
}

func (r *UDPRelayer) RelayUDP(relayer net.PacketConn, targetAddr, remoteAddr string) error {
	buf := r.natM.BufferPool.Get()
	defer r.natM.BufferPool.Put(buf)

	if r.inbound != nil {
		relayer = r.inbound.UdpConn(relayer)
	}
	// write data buffer range
	var fidx, tidx int

	var dstAddr address.Address
	if r.relayType == constant.UDPTunServerToSSServer {
		dstAddr = address.ParseAddress1(remoteAddr)
		copy(*buf, dstAddr)
		fidx = len(dstAddr)
	}

	n, srcAddr, err := relayer.ReadFrom((*buf)[fidx:])
	if err != nil {
		return err
	}

	tidx = n

	client := true
	switch r.relayType {
	case constant.UDPSSLocalToSSServer:
		dstAddr = address.ParseAddress1(targetAddr)
		fidx = 3
	case constant.UDPSSServerToUDPServer:
		dstAddr = address.ParseAddress3((*buf)[:tidx])
		targetAddr = dstAddr.String()
		fidx = len(dstAddr)
		client = false
	case constant.UDPTunServerToSSServer:
		tidx += fidx
		fidx = 0
	}
	dst := r.natM.Get(srcAddr.String())
	if dst == nil {
		dst, err = DialLocalUDP()
		if err != nil {
			return err
		}

		if r.outbound != nil {
			dst = r.outbound.UdpConn(dst)
		}

		if client {
			raddr, err := resolveUDPAddr(targetAddr)
			if err != nil {
				return err
			}
			dst = &SSPacketConn{PacketConn: dst, addr: raddr}
			targetAddr = raddr.String()
		}

		// symmetric nat
		dst = newSymmetricNATPacketConn(dst, targetAddr)

		r.natM.Add(srcAddr, relayer, dst, r.relayType-3)

		logx.Info("[%s] [%s] <-> [%s] <-> [%s], [%s]",
			color.BlueString("udp"),
			color.GreenString(srcAddr.String()),
			color.YellowString(relayer.LocalAddr().String()),
			color.RedString(dstAddr.String()),
			color.CyanString(targetAddr),
		)
	}

	if client {
		dst.WriteTo((*buf)[fidx:tidx], nil)
	} else {
		raddr, err := net.ResolveUDPAddr("udp", targetAddr)
		if err != nil {
			return err
		}
		dst.WriteTo((*buf)[fidx:tidx], raddr)
	}

	return nil
}

type symmetricNATPacketConn struct {
	net.PacketConn
	dst string
}

func newSymmetricNATPacketConn(pc net.PacketConn, dst string) *symmetricNATPacketConn {
	return &symmetricNATPacketConn{
		PacketConn: pc,
		dst:        dst,
	}
}

func (pc *symmetricNATPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		n, sender, err := pc.PacketConn.ReadFrom(p)
		// filter sender
		if sender != nil && sender.String() != pc.dst {
			logx.Warn("[UDP] symmetric NAT: packet isn't %s, so drop packet from %s", pc.dst, sender)
			continue
		}

		return n, sender, err
	}
}
