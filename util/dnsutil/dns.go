package dnsutil

import (
	"net/netip"
	"strings"

	"github.com/miekg/dns"
)

func TrimDomain(name string) string {
	return strings.TrimSuffix(name, ".")
}

func MsgToAddrs(reply *dns.Msg) []netip.Addr {
	var addrs []netip.Addr
	for _, answer := range reply.Answer {
		switch ans := answer.(type) {
		case *dns.A:
			if addr, ok := netip.AddrFromSlice(ans.A.To4()); ok {
				addrs = append(addrs, addr)
			}
		case *dns.AAAA:
			if addr, ok := netip.AddrFromSlice(ans.AAAA.To16()); ok {
				addrs = append(addrs, addr)
			}
		}
	}
	return addrs
}
