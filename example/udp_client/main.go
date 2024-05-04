package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/josexy/mini-ss/client"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "tun":
			udpTunMain()
		case "socks":
			socksMain()
		}
	} else {
		wg := sync.WaitGroup{}
		n := 100
		wg.Add(n)
		for i := 0; i < 50; i++ {
			go func() {
				defer wg.Done()
				udpTunMain()
			}()
		}
		for i := 0; i < 50; i++ {
			go func() {
				defer wg.Done()
				socksMain()
			}()
		}
		wg.Wait()
	}
}

// go run main.go tun 127.0.0.1:2003
func udpTunMain() {
	conn, err := transport.DialUDP(os.Args[2])
	if err != nil {
		panic(err)
	}
	done := make(chan error)
	defer conn.Close()

	go func() {
		buf := make([]byte, 65535)
		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			n, err := conn.Read(buf)
			if err != nil {
				done <- err
				return
			}
			log.Println(string(buf[:n]))
		}
	}()
	for i := 0; i < 10; i++ {
		conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		_, err = conn.Write([]byte("hello server " + time.Now().Format(time.DateTime)))
		if err != nil {
			log.Println(err)
			return
		}
	}
	err = <-done
	log.Println(err)
}

// go run main.go socks 127.0.0.1:10086 127.0.0.1:2003

func socksMain() {
	proxyCli := client.NewSocks5Client(os.Args[2])
	proxyCli.SetSocksAuth("123", "123")
	defer proxyCli.Close()
	conn, err := proxyCli.DialUDP(context.Background(), os.Args[3])
	if err != nil {
		panic(err)
	}
	done := make(chan error)
	go func() {
		buf := make([]byte, 65535)
		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 5))
			n, err := conn.Read(buf)
			if err != nil {
				done <- err
				return
			}
			log.Println(string(buf[:n]))
		}
	}()
	for i := 0; i < 10; i++ {
		conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
		_, err = conn.Write([]byte("hello server " + time.Now().Format(time.DateTime)))
		if err != nil {
			log.Println(err)
			return
		}
	}
	err = <-done
	log.Println(err)
}
