package hostsutil

import (
	"net/netip"
	"sync"

	"github.com/jaytaylor/go-hostsfile"
	"github.com/josexy/mini-ss/util/logger"
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
		logger.Logger.ErrorBy(err)
		return
	}
	for ip, hosts := range mp {
		for _, host := range hosts {
			if addr, err := netip.ParseAddr(ip); err == nil {
				logger.Logger.Infof("read hosts record: [%s]->[%s]", host, ip)
				hostsMap[host] = append(hostsMap[host], addr)
			}
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
