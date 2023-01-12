package hostsutil

import (
	"net/netip"
	"sync"

	"github.com/jaytaylor/go-hostsfile"
	"github.com/josexy/logx"
)

var (
	mu       sync.RWMutex
	once     sync.Once
	hostsMap map[string][]netip.Addr
)

func initHostsMap() {
	hostsMap = make(map[string][]netip.Addr)
	mp, err := hostsfile.ParseHosts(hostsfile.ReadHostsFile())
	if err != nil {
		logx.ErrorBy(err)
	}
	for ip, hosts := range mp {
		for _, host := range hosts {
			logx.Info("read hosts record: [%s]->[%s]", ip, host)
			hostsMap[host] = append(hostsMap[host], netip.MustParseAddr(ip))
		}
	}
}

func LookupIP(host string) netip.Addr {
	once.Do(func() {
		initHostsMap()
	})
	mu.RLock()
	defer mu.RUnlock()
	if ips, ok := hostsMap[host]; ok {
		return ips[0]
	}
	return netip.Addr{}
}
