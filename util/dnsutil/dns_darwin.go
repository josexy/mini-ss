package dnsutil

import (
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util"
	"github.com/miekg/dns"
)

func SetLocalDnsServer(addr string) {
	shell := `
function scutil_query {
key=$1
scutil <<EOT
open
get $key
d.show
close
EOT
}
function updateDNS {
    SERVICE_GUID=$(scutil_query State:/Network/Global/IPv4 | grep "PrimaryService" | awk '{print $3}')
    currentservice=$(scutil_query Setup:/Network/Service/$SERVICE_GUID | grep "UserDefinedName" | awk -F': ' '{print $2}')
    olddns=$(networksetup -getdnsservers "$currentservice")
	echo $olddns
    networksetup -setdnsservers "$currentservice" "` + addr + `"
}
function flushCache {
    dscacheutil -flushcache
    sudo killall -HUP mDNSResponder
}
updateDNS
flushCache
`
	if _, err := util.ExeShell(shell); err != nil {
		logx.ErrorBy(err)
	}
}

func UnsetLocalDnsServer() {
	shell := `
function scutil_query {
key=$1
scutil <<EOT
open
get $key
d.show
close
EOT
}
function updateDNS {
    SERVICE_GUID=$(scutil_query State:/Network/Global/IPv4 | grep "PrimaryService" | awk '{print $3}')
    currentservice=$(scutil_query Setup:/Network/Service/$SERVICE_GUID | grep "UserDefinedName" | awk -F': ' '{print $2}')
	networksetup -setdnsservers "$currentservice" empty
}
function flushCache {
    dscacheutil -flushcache
    sudo killall -HUP mDNSResponder
}
updateDNS
flushCache
`
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
