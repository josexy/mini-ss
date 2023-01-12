package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/socks"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "tun":
			udpTunMain()
		case "sock":
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

func udpTunMain() {
	conn, err := transport.DialUDP(os.Args[2])
	if err != nil {
		logx.FatalBy(err)
	}
	done := make(chan error)
	go func() {
		buf := make([]byte, 65535)
		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			n, err := conn.Read(buf)
			if err != nil {
				done <- err
				return
			}
			fmt.Println(string(buf[:n]))
		}
	}()
	for i := 0; i < 10; i++ {
		conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		_, err = conn.Write([]byte("hello server " + time.Now().String()))
		if err != nil {
			logx.ErrorBy(err)
			return
		}
	}
	err = <-done
	logx.ErrorBy(err)
}

// go run main.go sock 127.0.0.1:10086 127.0.0.1:2003

func socksMain() {
	proxyCli := socks.NewSocks5Client(os.Args[2])
	defer proxyCli.Close()

	conn, err := proxyCli.DialUDP(context.Background(), os.Args[3])
	if err != nil {
		logx.FatalBy(err)
	}
	done := make(chan error)
	go func() {
		buf := make([]byte, 65535)
		for {
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			n, err := conn.Read(buf)
			if err != nil {
				done <- err
				return
			}
			fmt.Println(string(buf[:n]))
		}
	}()
	for i := 0; i < 10; i++ {
		conn.SetWriteDeadline(time.Now().Add(time.Second * 2))
		_, err = conn.Write([]byte("hello server " + time.Now().String()))
		if err != nil {
			logx.ErrorBy(err)
			return
		}
	}
	err = <-done
	logx.ErrorBy(err)
}
