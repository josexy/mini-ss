package dnsutil

import (
	"net/netip"
	"strings"

	"github.com/miekg/dns"
)

func TrimDomain(name string) string {
	return strings.TrimSuffix(name, ".")
}

func MsgToIP(reply *dns.Msg) []netip.Addr {
	var ips []netip.Addr
	for _, answer := range reply.Answer {
		switch ans := answer.(type) {
		case *dns.A:
			if ip, ok := netip.AddrFromSlice(ans.A.To4()); ok {
				ips = append(ips, ip)
			}
		}
	}
	return ips
}
