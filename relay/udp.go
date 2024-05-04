package relay

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/transport"
)

const udpPacketTimeout = 30 * time.Second

var udpPool = bufferpool.NewBufferPool(bufferpool.MaxUdpBufferSize)

type (
	udpProxyReadFromSrcFunc func(net.Addr, []byte, int) ([]byte, net.Addr, error)
	udpProxyWriteToSrcFunc  func(net.Addr, []byte, int) ([]byte, error)
)

func IoCopyBidirectionalForPacket(srcConn, dstConn net.PacketConn, serverAddr string,
	udpReadFromSrc udpProxyReadFromSrcFunc, udpWriteToSrc udpProxyWriteToSrcFunc) error {

	var once sync.Once
	srcAddrCh := make(chan net.Addr, 1)
	errCh := make(chan error, 2)

	defer srcConn.Close()
	defer dstConn.Close()

	// srcAddr -> srcConn -> dstConn -> targetAddr
	go func() {
		buf := udpPool.Get()
		defer udpPool.Put(buf)
		for {
			srcConn.SetDeadline(time.Now().Add(udpPacketTimeout))
			n, srcAddr, err := srcConn.ReadFrom(*buf)
			if err != nil {
				// an error occurred on endpoint and the other endpoint needs to be close at the same time
				close(srcAddrCh)
				errCh <- err
				return
			}
			once.Do(func() { srcAddrCh <- srcAddr })

			b, targetAddr, err := udpReadFromSrc(srcAddr, *buf, n)
			if err != nil {
				continue
			}
			dstConn.WriteTo(b, targetAddr)
		}
	}()

	// targetAddr -> dstConn -> srcConn -> srcAddr
	go func() {
		// assume that the other endpoint has been 'established'
		// wait for another endpoint to send data
		var srcAddr net.Addr
		var ok bool
		srcAddr, ok = <-srcAddrCh
		if !ok {
			return
		}

		buf := udpPool.Get()
		defer udpPool.Put(buf)
		for {
			dstConn.SetDeadline(time.Now().Add(udpPacketTimeout))
			n, addr, err := dstConn.ReadFrom(*buf)
			if err != nil {
				errCh <- err
				return
			}
			// filter illegal data from the outside world
			if serverAddr != "" && addr != nil && addr.String() != serverAddr {
				continue
			}
			b, err := udpWriteToSrc(addr, *buf, n)
			if err != nil {
				continue
			}
			srcConn.WriteTo(b, srcAddr)
		}
	}()
	return <-errCh
}

type UDPDirectRelayer struct{}

func NewUDPDirectRelayer() *UDPDirectRelayer { return new(UDPDirectRelayer) }

func (r *UDPDirectRelayer) RelayToServer(conn net.PacketConn, remoteServerAddr string) error {
	var udpReadFromSrc udpProxyReadFromSrcFunc
	var udpWriteToSrc udpProxyWriteToSrcFunc

	if remoteServerAddr != "" {
		// Direct: the remote address customized
		targetAddr, err := net.ResolveUDPAddr("udp", remoteServerAddr)
		if err != nil {
			return err
		}
		// UDP Client -> [Relayer] -> UDP Server
		// {UDP data}
		udpReadFromSrc = func(_ net.Addr, buf []byte, n int) ([]byte, net.Addr, error) { return buf[:n], targetAddr, nil }
		// UDP Server -> [Relayer] -> UDP Client
		// {UDP data}
		udpWriteToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) { return buf[:n], nil }
	} else {
		// SOCKS5: the remote address come from UDP datagram
		// UDP Client -> Socks5 Client -> [Socks5 Server] -> UDP Server
		udpReadFromSrc = func(_ net.Addr, buf []byte, n int) ([]byte, net.Addr, error) {
			if n < 3 {
				return nil, nil, io.ErrShortBuffer
			}
			// buf: {0x00,0x00,0x00} {remote address} {UDP data}
			addr, err := address.ParseAddressFromBuffer(buf[3:n])
			if err != nil {
				return nil, nil, err
			}
			targetAddr, err := net.ResolveUDPAddr("udp", addr.String())
			if err != nil {
				return nil, nil, err
			}
			// return: {UDP data}
			return buf[3+len(addr) : n], targetAddr, nil
		}
		// UDP Server -> [Socks5 Server] -> Socks5 Client -> UDP Client
		udpWriteToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) {
			b := addrPool.Get()
			addr, err := address.ParseAddress(src.String(), *b)
			if err != nil {
				addrPool.Put(b)
				return nil, err
			}
			// buf: {UDP data}
			copy(buf[3+len(addr):], buf[:n])
			copy(buf[3:], addr)
			buf[0], buf[1], buf[2] = 0x00, 0x00, 0x00
			addrPool.Put(b)
			// return: {0x00,0x00,0x00} {remote address} {UDP data}
			return buf[:3+len(addr)+n], nil
		}
	}

	dstConn, err := transport.ListenLocalUDP(context.Background())
	if err != nil {
		return err
	}
	return IoCopyBidirectionalForPacket(conn, dstConn, "", udpReadFromSrc, udpWriteToSrc)
}

type ProxyUDPRelayer struct {
	proxyServerAddr   string
	inbound, outbound transport.UdpConnBound
}

func NewProxyUDPRelayer(proxyServerAddr string, inbound, outbound transport.UdpConnBound) *ProxyUDPRelayer {
	return &ProxyUDPRelayer{
		proxyServerAddr: proxyServerAddr,
		inbound:         inbound,
		outbound:        outbound,
	}
}

func (r *ProxyUDPRelayer) RelayToProxyServer(conn net.PacketConn, remoteServerAddr string) error {
	targetAddr, err := net.ResolveUDPAddr("udp", r.proxyServerAddr)
	if err != nil {
		return err
	}

	dstConn, err := transport.ListenLocalUDP(context.Background())
	if err != nil {
		return err
	}

	if r.outbound != nil {
		dstConn = r.outbound.UdpConn(dstConn)
	}

	var udpReadFromSrc udpProxyReadFromSrcFunc
	var udpWriteToSrc udpProxyWriteToSrcFunc

	if remoteServerAddr != "" {
		// SS: requires a remote server address
		// UDP Client -> [SS Client] -> SS Server -> UDP Server
		udpReadFromSrc = func(_ net.Addr, buf []byte, n int) ([]byte, net.Addr, error) {
			// buf: {UDP data}
			b := addrPool.Get()
			addr, err := address.ParseAddress(remoteServerAddr, *b)
			if err != nil {
				addrPool.Put(b)
				return nil, nil, err
			}
			copy(buf[len(addr):], buf[:n])
			copy(buf, addr)
			// return: {remote address} {UDP data}
			return buf[:n+len(addr)], targetAddr, nil
		}
		// UDP Server -> SS Server -> [SS Client] -> UDP Client
		udpWriteToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) {
			// buf: {remote address} {UDP data}
			b := addrPool.Get()
			addr, err := address.ParseAddress(src.String(), *b)
			if err != nil {
				addrPool.Put(b)
				return nil, err
			}
			// return: {UDP data}
			return buf[len(addr):n], nil
		}
	} else {
		// SOCKS5+SS: the remote address come from UDP datagram
		// UDP Client -> Socks5 Client -> [Socks5 Server + SS Client] -> SS Server -> UDP Server
		udpReadFromSrc = func(_ net.Addr, buf []byte, n int) ([]byte, net.Addr, error) {
			if n < 3 {
				return nil, nil, io.ErrShortBuffer
			}
			// buf:    {0x00,0x00,0x00} {remote address} {UDP data}
			// return: {remote address} {UDP data}
			return buf[3:n], targetAddr, nil
		}
		// UDP Server -> SS Server -> [SS Client + Socks5 Server] -> Socks5 Client -> UDP Client
		udpWriteToSrc = func(_ net.Addr, buf []byte, n int) ([]byte, error) {
			if n < 3 {
				return nil, io.ErrShortBuffer
			}
			// buf:    {remote address} {UDP data}
			// return: {0x00,0x00,0x00} {remote address} {UDP data}
			copy(buf[3:], buf[:n])
			buf[0], buf[1], buf[2] = 0x00, 0x00, 0x00
			return buf[:3+n], nil
		}
	}

	return IoCopyBidirectionalForPacket(conn, dstConn, r.proxyServerAddr, udpReadFromSrc, udpWriteToSrc)
}
