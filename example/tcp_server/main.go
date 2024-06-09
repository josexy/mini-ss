package main

import (
	"bytes"
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/server"
)

func main() {
	echoMain()
}

type echoSrv struct{}

func (echoSrv) ServeTCP(conn net.Conn) {
	log.Println(conn.RemoteAddr().String())
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
			break
		}
		xbuf := buf[:n]
		log.Printf("-> %d %s", n, string(xbuf))
		conn.Write(append(bytes.TrimRight(xbuf, "\r\n"), " <-- echo from server\n"...))
	}
}

// tcp client <-> quic client <-> quic server <-> tcp server

func echoMain() {
	srv := server.NewTcpServer(":10002", &echoSrv{}, server.Tcp)

	go func() {
		err := srv.Start(context.Background())
		log.Println("close server with err:", err)
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	<-interrupt

	srv.Close()
	time.Sleep(time.Second * 2)
}
