package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	srv := server.NewWsServer(":10086", nil, transport.DefaultWsOptions)
	srv.Handler = server.WsHandlerFunc(func(c net.Conn) {
		log.Println(c.LocalAddr(), c.RemoteAddr())
		c.Write([]byte("hello client"))
	})
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
