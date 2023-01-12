package transport

import (
	"net"
	"sync"
	"time"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/socks/constant"
)

type UdpNATMap struct {
	sync.RWMutex
	*bufferpool.BufferPool
	timeout time.Duration
	cache   map[string]net.PacketConn
}

func NewUdpNATMap(timeout time.Duration) *UdpNATMap {
	return &UdpNATMap{
		cache:      make(map[string]net.PacketConn),
		timeout:    timeout,
		BufferPool: bufferpool.NewBufferPool(constant.MaxUdpBufferSize),
	}
}

func (m *UdpNATMap) Get(srcAddr string) net.PacketConn {
	m.RLock()
	defer m.RUnlock()
	return m.cache[srcAddr]
}

func (m *UdpNATMap) Set(srcAddr string, targetConn net.PacketConn) {
	m.Lock()
	defer m.Unlock()

	m.cache[srcAddr] = targetConn
}

func (m *UdpNATMap) Del(srcAddr string) net.PacketConn {
	m.Lock()
	defer m.Unlock()
	if pc, ok := m.cache[srcAddr]; ok {
		delete(m.cache, srcAddr)
		return pc
	}
	return nil
}

func (m *UdpNATMap) Add(srcAddr net.Addr, dst, src net.PacketConn, op int) {
	m.Set(srcAddr.String(), src)

	go func() {
		// srcAddr <- dst <- src
		m.relay(srcAddr, dst, src, op)
		if conn := m.Del(srcAddr.String()); conn != nil {
			conn.Close()
		}
	}()
}

func (m *UdpNATMap) relay(srcAddr net.Addr, dst, src net.PacketConn, op int) error {
	bufferRead := m.BufferPool.Get()
	defer m.BufferPool.Put(bufferRead)

	for {
		src.SetReadDeadline(time.Now().Add(m.timeout))
		n, targetAddr, err := src.ReadFrom(*bufferRead)
		if err != nil {
			return err
		}
		// 1. ss-local 	-> client	[00 00 00 [target address] [payload data]]
		// 2. ss-server -> ss-local [[target address] [payload data]]
		// 3. udp-tun-local -> udp-client
		switch op {
		case constant.UDPSSLocalToSocksClient:
			// ss-server -> [ ss-local -> socks-client ]
			dst.WriteTo(append([]byte{0x00, 0x00, 0x00}, (*bufferRead)[:n]...), srcAddr)
		case constant.UDPSSServerToSSLocal:
			// udp-server -> [ ss-server -> ss-local ]
			// target address = udp server address
			addr := address.ParseAddress1(targetAddr.String())
			copy((*bufferRead)[len(addr):], (*bufferRead)[:n])
			copy((*bufferRead), addr)
			// payload data
			dst.WriteTo((*bufferRead)[:len(addr)+n], srcAddr)
		case constant.UDPTunServerToUDPClient:
			// ss-server -> [ udp-tun-local -> udp-client ]
			// need to remove the target address from ss-server
			// and write original udp packet data to udp client
			addr := address.ParseAddress1(targetAddr.String())
			dst.WriteTo(((*bufferRead)[len(addr):n]), srcAddr)
		}
	}
}

type UdpDirectNATMap struct{ *UdpNATMap }

func NewUdpDirectNATMap(timeout time.Duration) *UdpDirectNATMap {
	return &UdpDirectNATMap{
		UdpNATMap: NewUdpNATMap(timeout),
	}
}

func (m *UdpDirectNATMap) Add(srcAddr net.Addr, dst, src net.PacketConn) {
	m.Set(srcAddr.String(), src)

	go func() {
		m.directRelay(srcAddr, dst, src)
		if conn := m.Del(srcAddr.String()); conn != nil {
			conn.Close()
		}
	}()
}

func (m *UdpDirectNATMap) directRelay(srcAddr net.Addr, dst, src net.PacketConn) error {
	b := m.BufferPool.Get()
	defer m.BufferPool.Put(b)

	for {
		src.SetReadDeadline(time.Now().Add(m.timeout))
		n, _, err := src.ReadFrom(*b)
		if err != nil {
			return err
		}
		dst.WriteTo(append([]byte{0, 0, 0}, (*b)[:n]...), srcAddr)
	}
}
