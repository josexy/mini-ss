package tun

import (
	"fmt"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util"
)

func setTunIPAddress(name string, addr string, mtu int) (err error) {
	if mtu <= 0 {
		mtu = DefaultMTU
	}
	cmds := []string{
		fmt.Sprintf("ip link set dev %s mtu %d", name, mtu),
		fmt.Sprintf("ip address add %s dev %s", addr, name),
		fmt.Sprintf("ip link set dev %s up", name),
	}
	for _, cmd := range cmds {
		logx.Debug("%s", cmd)
		if err := util.ExeCmd(cmd); err != nil {
			return err
		}
	}
	return nil
}

func addTunNetRoutes(name string, routes []IPRoute) error {
	for _, route := range routes {
		if !route.Dest.IsValid() && !route.Gateway.IsValid() {
			continue
		}
		cmd := fmt.Sprintf("ip route add %s dev %s", route.Dest.String(), name)
		logx.Debug("%s", cmd)
		if err := util.ExeCmd(cmd); err != nil {
			return err
		}
	}
	return nil
}
