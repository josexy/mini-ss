// go:build !darwin && !linux

package transport

import (
	"errors"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/josexy/mini-ss/util"
)

func lookupLocalAddr(ifaceName string, network string, destination netip.Addr, port int) (netip.AddrPort, error) {
	// get interface by name
	ifaceObj, err := util.ResolveInterfaceByName(ifaceName)
	if err != nil {
		return netip.AddrPort{}, err
	}
	var addr netip.Addr
	switch network {
	case "udp4", "tcp4":
		addr = ifaceObj.PickIPv4Addr(destination)
	case "tcp6", "udp6":
		addr = ifaceObj.PickIPv6Addr(destination)
	default:
		if destination.IsValid() {
			if destination.Is4() {
				addr = ifaceObj.PickIPv4Addr(destination)
			} else {
				addr = ifaceObj.PickIPv6Addr(destination)
			}
		} else {
			addr = ifaceObj.PickIPv4Addr(destination)
		}
	}
	if !addr.IsValid() {
		return netip.AddrPort{}, errors.New("invalid ip address")
	}
	return netip.AddrPortFrom(addr, uint16(port)), nil
}

func bindIfaceToDialer(ifaceName string, dialer *net.Dialer, network string, destination netip.Addr) error {
	if !destination.IsGlobalUnicast() {
		return nil
	}

	local := uint64(0)
	if dialer.LocalAddr != nil {
		_, port, err := net.SplitHostPort(dialer.LocalAddr.String())
		if err == nil {
			local, _ = strconv.ParseUint(port, 10, 16)
		}
	}

	addrPort, err := lookupLocalAddr(ifaceName, network, destination, int(local))
	if err != nil {
		return err
	}

	var addr net.Addr
	if strings.HasPrefix(network, "tcp") {
		addr = &net.TCPAddr{
			IP:   addrPort.Addr().AsSlice(),
			Port: int(addrPort.Port()),
		}
	} else if strings.HasPrefix(network, "udp") {
		addr = &net.UDPAddr{
			IP:   addrPort.Addr().AsSlice(),
			Port: int(addrPort.Port()),
		}
	}
	dialer.LocalAddr = addr
	return nil
}

func bindIfaceToListenConfig(ifaceName string, _ *net.ListenConfig, network, address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		port = "0"
	}

	local, _ := strconv.ParseUint(port, 10, 16)

	addr, err := lookupLocalAddr(ifaceName, network, netip.Addr{}, int(local))
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}
