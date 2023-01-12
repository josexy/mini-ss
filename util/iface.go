package util

import (
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/josexy/logx"
)

var (
	once    sync.Once
	onceErr error
	mu      sync.Mutex
	res     map[string]*Interface
)

type Interface struct {
	Index        int
	Name         string
	Addrs        []netip.Prefix
	HardwareAddr net.HardwareAddr
	hasIPv4Addr  bool
}

func init() {
	ResolveAllInterfaces()
	if onceErr != nil {
		logx.FatalBy(onceErr)
	}
}

func (iface *Interface) PickIPv4Addr(destination netip.Addr) netip.Addr {
	return iface.pickIPAddr(destination, func(addr netip.Prefix) bool {
		return addr.Addr().Is4()
	})
}

func (iface *Interface) PickIPv6Addr(destination netip.Addr) netip.Addr {
	return iface.pickIPAddr(destination, func(addr netip.Prefix) bool {
		return addr.Addr().Is6()
	})
}

func (iface *Interface) pickIPAddr(destination netip.Addr, accept func(addr netip.Prefix) bool) netip.Addr {
	var fallback netip.Addr

	for _, addr := range iface.Addrs {
		if !accept(addr) {
			continue
		}

		// 169.254            link#18            UCS               en5
		if !fallback.IsValid() && !addr.Addr().IsLinkLocalUnicast() {
			fallback = addr.Addr()
			if !destination.IsValid() {
				break
			}
		}

		if destination.IsValid() && addr.Contains(destination) {
			return addr.Addr()
		}
	}

	return fallback
}

func ResolveInterfaceByIndex(index int) (*Interface, error) {
	mu.Lock()
	defer mu.Unlock()
	for _, iface := range res {
		if iface.Index == index {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface index %d not found", index)
}

func ResolveInterfaceByName(name string) (*Interface, error) {
	mu.Lock()
	defer mu.Unlock()
	if iface, ok := res[name]; ok {
		return iface, nil
	}
	return nil, fmt.Errorf("interface name %q not found", name)
}

func ResolveAllInterfaceName() []string {
	var list []string
	mu.Lock()
	defer mu.Unlock()
	for _, iface := range res {
		if iface.hasIPv4Addr {
			list = append(list, iface.Name)
		}
	}
	return list
}

func ResolveAllInterfaces() {
	once.Do(func() {
		res = make(map[string]*Interface)
		ifaces, err := net.Interfaces()
		if err != nil {
			onceErr = err
			return
		}
		// all interfaces
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()

			onceErr = err
			if err != nil {
				continue
			}
			ipNets := make([]netip.Prefix, 0, len(addrs))
			hasIPv4Addr := false
			for _, addr := range addrs {
				ipNet := netip.MustParsePrefix(addr.String())
				if ipNet.Addr().Is4() {
					hasIPv4Addr = true
				}
				ipNets = append(ipNets, ipNet)
			}
			res[iface.Name] = &Interface{
				Index:        iface.Index,
				Name:         iface.Name,
				Addrs:        ipNets,
				HardwareAddr: iface.HardwareAddr,
				hasIPv4Addr:  hasIPv4Addr,
			}
		}
	})
}
