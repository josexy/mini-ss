package tun

import (
	"fmt"
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util"
)

func setTunIPAddress(name string, addr string, mtu int) error {
	ip, _, _ := net.ParseCIDR(addr)
	if mtu <= 0 {
		mtu = DefaultMTU
	}
	cmd := fmt.Sprintf("ifconfig %s inet %s %s mtu %d up", name, addr, ip.String(), mtu)
	logx.Debug("%s", cmd)
	if err := util.ExeCmd(cmd); err != nil {
		return err
	}
	return nil
}

func addTunNetRoutes(name string, routes []IPRoute) error {
	for _, route := range routes {
		if !route.Dest.IsValid() && !route.Gateway.IsValid() {
			continue
		}
		cmd := fmt.Sprintf("route add -net %s %s", route.Dest.String(), route.Gateway)
		logx.Debug("%s", cmd)
		if err := util.ExeCmd(cmd); err != nil {
			return err
		}
	}
	return nil
}
