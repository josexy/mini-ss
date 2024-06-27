package resolver

import (
	"errors"
	"hash/adler32"
	"net/netip"
)

var errNoAvailableFakeIP = errors.New("no available fake ip")

type ipPool struct {
	hmin  uint32
	hmax  uint32
	bits  int
	flags []bool
}

func newIPPool(prefix netip.Prefix) (*ipPool, error) {
	if !prefix.IsValid() {
		return nil, errNoAvailableFakeIP
	}
	var hmin, hmax uint32
	var base uint32 = ipToInt(prefix.Masked().Addr())
	var mask uint32 = (0xFFFFFFFF << (32 - prefix.Bits())) & 0xFFFFFFFF
	if mask == 0xFFFFFFFF {
		return nil, errNoAvailableFakeIP
	}

	last := base | (^mask & 0xFFFFFFFF)
	if mask == 0xFFFFFFFE {
		hmin = base
		hmax = last
	} else {
		hmin = base + 1
		hmax = last - 1
	}
	hostn := hmax - hmin + 1
	return &ipPool{
		hmin:  hmin,
		hmax:  hmax,
		bits:  prefix.Masked().Bits(),
		flags: make([]bool, hostn),
	}, nil
}

func (pool *ipPool) Capacity() int { return cap(pool.flags) }

func (pool *ipPool) IPMin() netip.Addr { return intToIP(pool.hmin) }

func (pool *ipPool) IPMax() netip.Addr { return intToIP(pool.hmax) }

func (pool *ipPool) Bits() int { return pool.bits }

func (pool *ipPool) Available() int {
	var count int
	for _, used := range pool.flags {
		if !used {
			count++
		}
	}
	return count
}

func (pool *ipPool) index(ip netip.Addr) int {
	value := ipToInt(ip)
	if value >= pool.hmin && value <= pool.hmax {
		return int(value - pool.hmin)
	}
	return -1
}

func (pool *ipPool) Contains(ip netip.Addr) bool {
	return pool.index(ip) != -1
}

func (pool *ipPool) IsAvailable(ip netip.Addr) bool {
	if index := pool.index(ip); index != -1 {
		return !pool.flags[index]
	}
	return false
}

func (pool *ipPool) allocateFor(ip netip.Addr) (netip.Addr, bool) {
	index := pool.index(ip)
	if index != -1 {
		pool.flags[index] = true
		return ip, true
	}
	// choose next available ip
	for i, used := range pool.flags {
		if !used {
			index = i
			break
		}
	}
	if index == -1 {
		return netip.Addr{}, false
	}
	pool.flags[index] = true
	return intToIP(pool.hmin + uint32(index)), true
}

func (pool *ipPool) Release(ip netip.Addr) bool {
	if index := pool.index(ip); index != -1 {
		pool.flags[index] = false
		return true
	}
	return false
}

func (pool *ipPool) Allocate(host string) (ip netip.Addr, err error) {
	index := adler32.Checksum([]byte(host)) % uint32(cap(pool.flags))
	if pool.flags[index] {
		for i, used := range pool.flags {
			if !used {
				index = uint32(i)
				break
			}
		}
	}

	if pool.flags[index] {
		err = errNoAvailableFakeIP
		return
	}
	pool.flags[index] = true
	ip = intToIP(pool.hmin + index)
	return
}

func intToIP(v uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func ipToInt(ip netip.Addr) uint32 {
	v := ip.As4()
	return (uint32(v[0]) << 24) | (uint32(v[1]) << 16) | (uint32(v[2]) << 8) | uint32(v[3])
}
