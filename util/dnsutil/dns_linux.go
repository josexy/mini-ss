package dnsutil

import (
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util"
	"github.com/miekg/dns"
)

var oldDnsValue string

func SetLocalDnsServer(addr string) {
	shell := `
olddns=$(cat /etc/resolv.conf |grep nameserver| cut -d ' ' -f 2)
echo $olddns
echo "nameserver ` + addr + `" | sudo tee /etc/resolv.conf
`
	if out, err := util.ExeShell(shell); err == nil {
		oldDnsValue = out
	} else {
		logx.ErrorBy(err)
	}
}

func UnsetLocalDnsServer() {
	shell := `
echo "nameserver ` + oldDnsValue + `" | sudo tee /etc/resolv.conf
`
	if out, err := util.ExeShell(shell); err == nil {
		oldDnsValue = out
	}
}

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
