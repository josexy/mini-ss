package util

import (
	"net"
	"net/netip"
	"testing"
)

func BenchmarkNetip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		prefix := netip.MustParsePrefix("192.168.1.2/16")
		_ = prefix.Contains(netip.MustParseAddr("192.168.1.222"))
	}
}

func BenchmarkNet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, subnet, _ := net.ParseCIDR("192.168.1.2/16")
		_ = subnet.Contains(net.ParseIP("192.168.1.222"))
	}
}

func BenchmarkNetipToString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addr := netip.MustParseAddr("192.168.1.2")
		_ = addr.String()
	}
}

func BenchmarkNetToString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ip := net.ParseIP("192.168.1.2")
		_ = ip.String()
	}
}

func BenchmarkNetipMaskSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		addr := netip.MustParsePrefix("192.168.1.2/10")
		_ = addr.Masked().Bits()
	}
}

func BenchmarkNetMaskSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, subnet, _ := net.ParseCIDR("192.168.1.2/10")
		_, _ = subnet.Mask.Size()
	}
}
