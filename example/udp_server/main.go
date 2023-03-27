package main

import (
	"log"
	"net"

	"github.com/josexy/mini-ss/relay"
)

func main() {
	go relayer()
	server()
}

func relayer() {
	addr, err := net.ResolveUDPAddr("udp", ":2003")
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	targetAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2004")
	err = relay.RelayUDPWithNatmap(conn,
		func(srcAddr net.Addr, in []byte, n int) ([]byte, *net.UDPAddr, error) {
			return in[:n], targetAddr, nil
		}, func(src net.Addr, in []byte, n int) ([]byte, error) {
			return in[:n], nil
		})
	log.Println(err)
}

func server() {
	addr, err := net.ResolveUDPAddr("udp", ":2004")
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	log.Printf("local addr: %v\n", conn.LocalAddr())

	buf := make([]byte, 65535)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}
		conn.WriteTo(buf[:n], addr)
		log.Println(addr, "->", string(buf[:n]))
	}
}
