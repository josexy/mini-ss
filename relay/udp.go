package relay

import (
	"net"
	"sync"
	"time"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/transport"
)

var udpPool = bufferpool.NewBufferPool(constant.MaxUdpBufferSize)

type (
	srcToDstFn func(src net.Addr, in []byte, n int) ([]byte, *net.UDPAddr, error)
	dstToSrcFn func(src net.Addr, in []byte, n int) ([]byte, error)
)

func RelayUDP(src, dst net.PacketConn, targetAddr string, srcToDst srcToDstFn, dstToSrc dstToSrcFn) error {
	var once sync.Once
	srcAddrCh := make(chan net.Addr, 1)
	errCh := make(chan error, 2)

	defer src.Close()
	defer dst.Close()

	// srcAddr -> relayer -> targetAddr
	go func() {
		buf := udpPool.Get()
		defer udpPool.Put(buf)
		for {
			src.SetDeadline(time.Now().Add(constant.UdpTimeout))
			n, srcAddr, err := src.ReadFrom(*buf)
			if err != nil {
				// an error occurred on endpoint and the other endpoint needs to be close
				close(srcAddrCh)
				errCh <- err
				return
			}
			once.Do(func() { srcAddrCh <- srcAddr })

			b, targetAddr, err := srcToDst(srcAddr, *buf, n)
			if err != nil {
				continue
			}
			dst.WriteTo(b, targetAddr)
		}
	}()

	// targetAddr -> relayer -> srcAddr
	go func() {
		// waiting for another endpoint to send data
		var srcAddr net.Addr
		var ok bool
		srcAddr, ok = <-srcAddrCh
		if !ok {
			return
		}

		buf := udpPool.Get()
		defer udpPool.Put(buf)
		for {
			dst.SetDeadline(time.Now().Add(constant.UdpTimeout))
			n, addr, err := dst.ReadFrom(*buf)
			// filter illegal data from the outside world
			if targetAddr != "" && addr != nil && addr.String() != targetAddr {
				continue
			}
			if err != nil {
				errCh <- err
				return
			}
			b, err := dstToSrc(addr, *buf, n)
			if err != nil {
				continue
			}
			src.WriteTo(b, srcAddr)
		}
	}()
	return <-errCh
}

func RelayUDPWithNatmap(src net.PacketConn, srcToDst srcToDstFn, dstToSrc dstToSrcFn) error {
	nm := struct {
		sync.RWMutex
		cache map[string]net.PacketConn
	}{cache: make(map[string]net.PacketConn)}

	// targetAddr -> relayer -> srcAddr
	handleDstToRelayer := func(srcAddr net.Addr, dstConn net.PacketConn, targetAddr string) {
		buf := udpPool.Get()
		defer func() {
			udpPool.Put(buf)
			nm.Lock()
			delete(nm.cache, srcAddr.String())
			nm.Unlock()
			dstConn.Close()
		}()

		for {
			dstConn.SetDeadline(time.Now().Add(constant.UdpTimeout))
			n, addr, err := dstConn.ReadFrom(*buf)

			// filter illegal data from the outside world
			if targetAddr != "" && addr != nil && addr.String() != targetAddr {
				continue
			}
			if err != nil {
				return
			}
			b, err := dstToSrc(addr, *buf, n)
			if err != nil {
				continue
			}
			src.WriteTo(b, srcAddr)
		}
	}

	buf := udpPool.Get()
	defer udpPool.Put(buf)

	// srcAddr  -> relayer   -> targetAddr
	for {
		n, srcAddr, err := src.ReadFrom(*buf)
		if err != nil {
			return err
		}

		b, targetAddr, err := srcToDst(srcAddr, *buf, n)
		if err != nil {
			continue
		}

		nm.RLock()
		dstConn := nm.cache[srcAddr.String()]
		nm.RUnlock()

		if dstConn == nil {
			dstConn, err = transport.ListenLocalUDP()
			if err != nil {
				continue
			}

			nm.Lock()
			nm.cache[srcAddr.String()] = dstConn
			nm.Unlock()

			go handleDstToRelayer(srcAddr, dstConn, targetAddr.String())
		}

		dstConn.WriteTo(b, targetAddr)
	}
}

type UDPDirectRelayer struct{}

func NewUDPDirectRelayer() *UDPDirectRelayer { return new(UDPDirectRelayer) }

func (r *UDPDirectRelayer) RelayDirectUDP(relayer net.PacketConn, remoteAddr string) error {
	var srcToDst srcToDstFn
	var dstToSrc dstToSrcFn

	if remoteAddr != "" {
		targetAddr, err := net.ResolveUDPAddr("udp", remoteAddr)
		if err != nil {
			return err
		}
		// requires a remote server address
		// {UDP data}
		srcToDst = func(_ net.Addr, buf []byte, n int) ([]byte, *net.UDPAddr, error) { return buf[:n], targetAddr, nil }
		// {UDP data}
		dstToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) { return buf[:n], nil }
	} else {
		// SOCKS5: the remote address come from UDP datagram
		srcToDst = func(_ net.Addr, buf []byte, n int) ([]byte, *net.UDPAddr, error) {
			// {0x00,0x00,0x00} {remote address} {UDP data}
			addr := address.ParseAddress3(buf[3:n])
			targetAddr, err := net.ResolveUDPAddr("udp", addr.String())
			if err != nil {
				return nil, nil, err
			}
			// {UDP data}
			return buf[3+len(addr) : n], targetAddr, nil
		}
		dstToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) {
			// {UDP data}
			addr := address.ParseAddress1(src.String())
			copy(buf[3+len(addr):], buf[:n])
			copy(buf[3:], addr)
			buf[0], buf[1], buf[2] = 0x00, 0x00, 0x00
			// {0x00,0x00,0x00} {remote address} {UDP data}
			return buf[:3+len(addr)+n], nil
		}
	}

	dstConn, err := transport.ListenLocalUDP()
	if err != nil {
		return err
	}
	return RelayUDP(relayer, dstConn, "", srcToDst, dstToSrc)
}

type DstUDPRelayer struct {
	DstAddr string
	*UDPRelayer
}

type UDPRelayer struct{ outbound transport.UdpConnBound }

func NewUDPRelayer(outbound transport.UdpConnBound) *UDPRelayer {
	return &UDPRelayer{outbound: outbound}
}

func (r *UDPRelayer) RelayLocalToServer(relayer net.PacketConn, serverAddr, remoteAddr string) error {
	targetAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return err
	}

	dstConn, err := transport.ListenLocalUDP()
	if err != nil {
		return err
	}

	if r.outbound != nil {
		dstConn = r.outbound.UdpConn(dstConn)
	}

	var srcToDst srcToDstFn
	var dstToSrc dstToSrcFn

	if remoteAddr != "" {
		// requires a remote server address
		addr := address.ParseAddress1(remoteAddr)
		srcToDst = func(_ net.Addr, buf []byte, n int) ([]byte, *net.UDPAddr, error) {
			// {UDP data}
			copy(buf[len(addr):], buf[:n])
			copy(buf, addr)
			// {remote address} {UDP data}
			return buf[:n+len(addr)], targetAddr, nil
		}
		dstToSrc = func(src net.Addr, buf []byte, n int) ([]byte, error) {
			// {remote address} {UDP data}
			addr := address.ParseAddress1(src.String())
			// {UDP data}
			return buf[len(addr):n], nil
		}
	} else {
		// SOCKS5: the remote address come from UDP datagram
		srcToDst = func(_ net.Addr, buf []byte, n int) ([]byte, *net.UDPAddr, error) {
			// {0x00,0x00,0x00} {remote address} {UDP data}
			// {remote address} {UDP data}
			return buf[3:n], targetAddr, nil
		}
		dstToSrc = func(_ net.Addr, buf []byte, n int) ([]byte, error) {
			// {remote address} {UDP data}
			// {0x00,0x00,0x00} {remote address} {UDP data}
			return append([]byte{0x00, 0x00, 0x00}, buf[:n]...), nil
		}
	}

	return RelayUDP(relayer, dstConn, serverAddr, srcToDst, dstToSrc)
}

type NatmapUDPRelayer struct{ inbound transport.UdpConnBound }

func NewNatmapUDPRelayer(inbound transport.UdpConnBound) *NatmapUDPRelayer {
	return &NatmapUDPRelayer{inbound: inbound}
}

func (r *NatmapUDPRelayer) RelayServerToRemote(relayer net.PacketConn) error {
	if r.inbound != nil {
		relayer = r.inbound.UdpConn(relayer)
	}

	srcToDst := func(_ net.Addr, buf []byte, n int) ([]byte, *net.UDPAddr, error) {
		// {remote address} {UDP data}
		addr := address.ParseAddress3(buf[:n])
		targetAddr, err := net.ResolveUDPAddr("udp", addr.String())
		if err != nil {
			return nil, nil, err
		}
		// {UDP data}
		return buf[len(addr):n], targetAddr, nil
	}

	dstToSrc := func(src net.Addr, buf []byte, n int) ([]byte, error) {
		// {UDP data}
		addr := address.ParseAddress1(src.String())
		copy(buf[len(addr):], buf[:n])
		copy(buf, addr)
		// {remote address} {UDP data}
		return buf[:len(addr)+n], nil
	}

	return RelayUDPWithNatmap(relayer, srcToDst, dstToSrc)
}
