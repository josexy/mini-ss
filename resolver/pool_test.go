package resolver

import (
	"math/rand"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

var letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func TestIPPool_AllocateAndRelease(t *testing.T) {
	testFn := func(cidr string) {
		pool, err := newIPPool(cidr)
		assert.Nil(t, err)

		t.Logf("new ip pool for cidr: %s", cidr)
		t.Logf("ip pool ip available: %d, capacity: %d, bits: %d", pool.Available(), pool.Capacity(), pool.Bits())
		t.Logf("ip pool ip range: [%s - %s]", pool.IPMin(), pool.IPMax())

		assert.True(t, pool.Contains(pool.IPMin()))
		assert.True(t, pool.Contains(pool.IPMax()))
		assert.False(t, pool.Contains(pool.IPMin().Prev()))
		assert.False(t, pool.Contains(pool.IPMax().Next()))

		for min := pool.hmin; min <= pool.hmax; min++ {
			ip := intToIP(min)
			assert.True(t, pool.Contains(ip))
		}

		ips := make([]netip.Addr, pool.Capacity())

		// allocate
		for i := 0; i < pool.Capacity(); i++ {
			assert.Equal(t, pool.Available()+i, pool.Capacity())
			ip, err := pool.Allocate(randStr(rand.Intn(25) + 5))
			assert.Nil(t, err)
			assert.False(t, pool.IsAvailable(ip))
			assert.True(t, pool.Contains(ip))
			ips[i] = ip
		}
		assert.Equal(t, pool.Available(), 0)

		// release
		for i := pool.Capacity(); i > 0; i-- {
			assert.Equal(t, pool.Available()+i, pool.Capacity())
			ip := ips[i-1]
			assert.True(t, pool.Release(ip))
			assert.True(t, pool.IsAvailable(ip))
			assert.True(t, pool.Contains(ip))
			ips = ips[:i-1]
		}
		assert.Equal(t, pool.Available(), pool.Capacity())
	}
	testFn2 := func(cidr string) {
		pool, err := newIPPool(cidr)
		assert.Nil(t, err)
		prefix := netip.MustParsePrefix(cidr)
		ip, ok := pool.allocateFor(prefix.Addr())
		assert.True(t, ok)
		assert.True(t, pool.Contains(ip))
		t.Logf("=== allocate for: %s, actual: %s, remain: %d", prefix.Addr().String(), ip.String(), pool.Available())

		ip2, ok := pool.allocateFor(ip.Next())
		assert.True(t, ok)
		assert.True(t, pool.Contains(ip2))
		t.Logf("=== allocate for: %s, actual: %s, remain: %d", ip.Next().String(), ip2.String(), pool.Available())
	}
	cidrs := []string{
		"10.0.0.2/31",
		"10.0.0.2/30",
		"10.0.0.4/30",
		"10.2.99.14/22",
		"10.2.88.25/18",
		"254.244.240.25/16",
	}
	for _, cidr := range cidrs {
		testFn(cidr)
		testFn2(cidr)
	}
}
