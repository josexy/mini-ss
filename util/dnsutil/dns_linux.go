package dnsutil

import (
	"net"

	"github.com/miekg/dns"
)

func GetLocalDnsList() []string {
	config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	var localDnsServer []string
	if err == nil {
		for _, server := range config.Servers {
			localDnsServer = append(localDnsServer, net.JoinHostPort(server, config.Port))
		}
	}
	return localDnsServer
}
