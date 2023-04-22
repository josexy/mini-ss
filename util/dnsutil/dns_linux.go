package dnsutil

import (
	"net"

	"github.com/josexy/mini-ss/util"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/miekg/dns"
)

var oldDnsValue string

func SetLocalDnsServer(addr string) {
	shell := `
olddns="$(cat /etc/resolv.conf)"
echo "nameserver ` + addr + `" | sudo tee /etc/resolv.conf 1>/dev/null 2>&1
echo "$olddns"
`
	if out, err := util.ExeShell(shell); err == nil {
		oldDnsValue = out
	} else {
		logger.Logger.ErrorBy(err)
	}
}

func UnsetLocalDnsServer() {
	shell := `
cat << EOF | sudo tee /etc/resolv.conf
` +
		oldDnsValue + `

EOF`
	util.ExeShell(shell)
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
