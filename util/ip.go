package util

import (
	"net"
	"net/netip"
	"strconv"
)

// SplitHostPort split address into host and port
// 127.0.0.1:80
// 127.0.0.1
// example.com:80
// example.com
func SplitHostPort(addr string) (host, port string) {
	var err error
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		host = addr
		return
	}
	return
}

func IntToIP(v uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func IPToInt(ip netip.Addr) uint32 {
	v := ip.As4()
	return (uint32(v[0]) << 24) | (uint32(v[1]) << 16) | (uint32(v[2]) << 8) | uint32(v[3])
}

func MustStringToInt(s string) int {
	v, _ := strconv.ParseInt(s, 10, 32)
	return int(v)
}
