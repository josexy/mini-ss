package resolver

import (
	"hash/adler32"
	"net/netip"
)

type fakeIPPool struct {
	base  uint32
	space uint32
	flags []bool
}

func newFakeIPPool(cidr string) *fakeIPPool {
	subnet := netip.MustParsePrefix(cidr)
	ip := subnet.Addr()
	base := ipToInt(ip) + 1

	// the number of hosts
	var mask uint32 = (0xFFFFFFFF << (32 - subnet.Bits())) & 0xFFFFFFFF
	max := base + ^mask

	space := max - base
	if space > 0x3ffff {
		space = 0x3ffff
	}
	flags := make([]bool, space)

	// ip is used by tun
	index := ipToInt(ip) - base
	if index < space {
		flags[index] = true
	}

	return &fakeIPPool{
		base:  base,
		space: space,
		flags: flags,
	}
}

func (pool *fakeIPPool) Capacity() int {
	return int(pool.space)
}

func (pool *fakeIPPool) Contains(ip netip.Addr) bool {
	index := ipToInt(ip) - pool.base
	return index < pool.space
}

func (pool *fakeIPPool) Release(ip netip.Addr) {
	index := ipToInt(ip) - pool.base
	if index < pool.space {
		pool.flags[index] = false
	}
}

func (pool *fakeIPPool) Alloc(host string) netip.Addr {
	index := adler32.Checksum([]byte(host)) % pool.space
	if pool.flags[index] {
		for i, used := range pool.flags {
			if !used {
				index = uint32(i)
				break
			}
		}
	}

	if pool.flags[index] {
		return netip.Addr{}
	}
	pool.flags[index] = true
	return intToIP(pool.base + index)
}

func intToIP(v uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func ipToInt(ip netip.Addr) uint32 {
	v := ip.As4()
	return (uint32(v[0]) << 24) | (uint32(v[1]) << 16) | (uint32(v[2]) << 8) | uint32(v[3])
}
