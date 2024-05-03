package relay

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
)

type NatmapUDPRelayer struct {
	sync.RWMutex
	cache    map[string]net.PacketConn
	inbound  transport.UdpConnBound
	outbound transport.UdpConnBound
}

func NewNatmapUDPRelayer(inbound, outbound transport.UdpConnBound) *NatmapUDPRelayer {
	return &NatmapUDPRelayer{
		cache:    make(map[string]net.PacketConn),
		inbound:  inbound,
		outbound: outbound,
	}
}

func (r *NatmapUDPRelayer) DirectRelayToServer(conn net.PacketConn, remoteAddr string) error {
	targetAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
	if err != nil {
		return err
	}

	if r.inbound != nil {
		conn = r.inbound.UdpConn(conn)
	}

	var udpReadFromSrc udpProxyReadFromSrcFunc
	var udpWriteToSrc udpProxyWriteToSrcFunc

	// UDP Client -> [Relayer] -> UDP Server
	udpReadFromSrc = func(_ net.Addr, buf []byte, n int) ([]byte, net.Addr, error) { return buf[:n], targetAddr, nil }
	// UDP Server -> [Relayer] -> UDP Client
	udpWriteToSrc = func(_ net.Addr, buf []byte, n int) ([]byte, error) { return buf[:n], nil }

	return r.relayUDP(conn, udpReadFromSrc, udpWriteToSrc)
}

func (r *NatmapUDPRelayer) RelayToServer(conn net.PacketConn) error {
	if r.inbound != nil {
		conn = r.inbound.UdpConn(conn)
	}

	var udpReadFromSrc udpProxyReadFromSrcFunc
	var udpWriteToSrc udpProxyWriteToSrcFunc

	// SOCKS5+SS: the remote address come from UDP datagram
	// UDP Client -> Socks5 Client -> Socks5 Server + SS Client -> [SS Server] -> UDP Server
	udpReadFromSrc = func(_ net.Addr, buf []byte, n int) ([]byte, net.Addr, error) {
		// buf: {remote address} {UDP data}
		addr, err := address.ParseAddressFromBuffer(buf[:n])
		if err != nil {
			return nil, nil, err
		}
		targetAddr, err := net.ResolveUDPAddr("udp", addr.String())
		if err != nil {
			return nil, nil, err
		}
		// return: {UDP data}
		return buf[len(addr):n], targetAddr, nil
	}
	// UDP Server -> [SS Server] -> SS Client + Socks5 Server -> Socks5 Client -> UDP Client
	udpWriteToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) {
		b := addrPool.Get()
		// buf: {UDP data}
		addr, err := address.ParseAddress(src.String(), *b)
		if err != nil {
			addrPool.Put(b)
			return nil, err
		}
		copy(buf[len(addr):], buf[:n])
		copy(buf, addr)
		addrPool.Put(b)
		// return: {remote address} {UDP data}
		return buf[:len(addr)+n], nil
	}

	return r.relayUDP(conn, udpReadFromSrc, udpWriteToSrc)
}

func (r *NatmapUDPRelayer) relayUDP(conn net.PacketConn, udpReadFromSrc udpProxyReadFromSrcFunc, udpWriteToSrc udpProxyWriteToSrcFunc) error {

	defer func() {
		// close all packet conns
		conns := make([]net.PacketConn, 0, len(r.cache))
		for _, conn := range r.cache {
			conns = append(conns, conn)
		}
		for _, conn := range conns {
			conn.Close()
		}
	}()

	// server -> proxy -> client
	handleDstToRelayer := func(srcAddr net.Addr, dstConn net.PacketConn, targetAddr string) {
		buf := udpPool.Get()
		defer func() {
			udpPool.Put(buf)
			dstConn.Close()
			r.Lock()
			delete(r.cache, srcAddr.String())
			r.Unlock()
		}()

		for {
			dstConn.SetDeadline(time.Now().Add(udpPacketTimeout))
			n, addr, err := dstConn.ReadFrom(*buf)
			if err != nil {
				logger.Logger.ErrorBy(err)
				return
			}
			// filter illegal data from the outside world
			if targetAddr != "" && addr != nil && addr.String() != targetAddr {
				continue
			}
			b, err := udpWriteToSrc(addr, *buf, n)
			if err != nil {
				logger.Logger.ErrorBy(err)
				continue
			}
			conn.WriteTo(b, srcAddr)
		}
	}

	buf := udpPool.Get()
	defer udpPool.Put(buf)

	// client -> proxy -> server
	for {
		n, srcAddr, err := conn.ReadFrom(*buf)
		if err != nil {
			logger.Logger.ErrorBy(err)
			return err
		}

		b, targetAddr, err := udpReadFromSrc(srcAddr, *buf, n)
		if err != nil {
			logger.Logger.ErrorBy(err)
			continue
		}

		r.RLock()
		dstConn, ok := r.cache[srcAddr.String()]
		r.RUnlock()

		if !ok || dstConn == nil {
			dstConn, err = transport.ListenLocalUDP(context.Background())
			if err != nil {
				logger.Logger.ErrorBy(err)
				continue
			}

			if r.outbound != nil {
				dstConn = r.outbound.UdpConn(dstConn)
			}

			r.Lock()
			r.cache[srcAddr.String()] = dstConn
			r.Unlock()

			go handleDstToRelayer(srcAddr, dstConn, targetAddr.String())
		}

		dstConn.WriteTo(b, targetAddr)
	}
}
