package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/relay"
)

func main() {
	go server()
	relayer()
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

	log.Printf("local addr: %v\n", conn.LocalAddr())
	nmUdpRelayer := relay.NewNatmapUDPRelayer(nil, nil)
	go func() {
		err := nmUdpRelayer.DirectRelayToServer(conn, "localhost:2004")
		fmt.Println(err)
	}()

	inter := make(chan os.Signal, 1)
	signal.Notify(inter, syscall.SIGINT)
	<-inter
	conn.Close()
	time.Sleep(time.Second * 2)
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
