package tun

import (
	"fmt"
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util"
)

func setTunIPAddress(name string, addr string, mtu int) (err error) {
	ip, subnet, _ := net.ParseCIDR(addr)
	cmd := fmt.Sprintf("netsh interface ip set address \"%s\" static %s %s none", name, ip.String(), net.IP(subnet.Mask).String())
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
		var mask uint32 = (0xFFFFFFFF << (32 - route.Dest.Bits())) & 0xFFFFFFFF
		cmd := fmt.Sprintf("route add %s mask %s %s",
			route.Dest.Addr().String(),
			util.IntToIP(mask).String(),
			route.Gateway)
		logx.Debug("%s", cmd)
		if err := util.ExeCmd(cmd); err != nil {
			return err
		}
	}
	return nil
}
