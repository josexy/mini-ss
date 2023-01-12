package main

import (
	"fmt"
	"net"

	"github.com/josexy/logx"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":2003")
	if err != nil {
		logx.FatalBy(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logx.ErrorBy(err)
		return
	}
	defer conn.Close()

	logx.Info("remote addr: %v", conn.RemoteAddr())
	logx.Info("local addr: %v", conn.LocalAddr())

	buf := make([]byte, 65535)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			logx.ErrorBy(err)
			return
		}
		conn.WriteTo(buf[:n], addr)
		fmt.Println(addr, "->", string(buf[:n]))
	}
}
