package transport

import (
	"net"
	"net/netip"
	"syscall"

	"github.com/josexy/mini-ss/util"
	"golang.org/x/sys/unix"
)

type controlFn = func(network, address string, c syscall.RawConn) error

func bindControl(ifaceIdx int, chain controlFn) controlFn {
	return func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && chain != nil {
				err = chain(network, address, c)
			}
		}()

		ipStr, _, err := net.SplitHostPort(address)
		if err == nil {
			if ip, err := netip.ParseAddr(ipStr); err == nil && !ip.IsGlobalUnicast() {
				return err
			}
		}

		// bind interface to socket
		// the outbound local address was changed to interface address
		var innerErr error
		err = c.Control(func(fd uintptr) {
			switch network {
			case "tcp4", "udp4":
				innerErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, ifaceIdx)
			case "tcp6", "udp6":
				innerErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, ifaceIdx)
			}
		})

		if innerErr != nil {
			err = innerErr
		}

		return
	}
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, _ string, _ netip.Addr) error {
	ifaceObj, err := util.ResolveInterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	dialer.Control = bindControl(ifaceObj.Index, dialer.Control)
	return nil
}

func bindIfaceToListenConfig(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	ifaceObj, err := util.ResolveInterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	lc.Control = bindControl(ifaceObj.Index, lc.Control)
	return address, nil
}
