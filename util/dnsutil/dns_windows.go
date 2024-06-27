package dnsutil

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func GetLocalDnsList() []string {
	l := uint32(20000)
	b := make([]byte, l)

	if err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_PREFIX, 0,
		(*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])), &l); err != nil {
		return nil
	}
	var addresses []*windows.IpAdapterAddresses
	for addr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&b[0])); addr != nil; addr = addr.Next {
		addresses = append(addresses, addr)
	}

	resolvers := map[string]struct{}{}
	for _, addr := range addresses {
		for next := addr.FirstUnicastAddress; next != nil; next = next.Next {
			if addr.OperStatus != windows.IfOperStatusUp {
				continue
			}
			if next.Address.IP() != nil {
				for dnsServer := addr.FirstDnsServerAddress; dnsServer != nil; dnsServer = dnsServer.Next {
					ip := dnsServer.Address.IP()
					if ip.IsMulticast() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
						continue
					}
					if ip.To16() != nil && strings.HasPrefix(ip.To16().String(), "fec0:") {
						continue
					}
					resolvers[ip.String()] = struct{}{}
				}
				break
			}
		}
	}

	servers := []string{}
	for server := range resolvers {
		servers = append(servers, server)
	}
	return servers
}
