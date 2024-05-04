package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	srv := server.NewGrpcServer(":10086", nil, transport.DefaultGrpcOptions)
	srv.Handler = server.GrpcHandler(server.GrpcHandlerFunc(func(c net.Conn) {
		log.Println(c.LocalAddr(), c.RemoteAddr())
		conn, err := net.Dial("tcp", "www.baidu.com:80")
		if err != nil {
			log.Println(err)
			return
		}
		relay.IoCopyBidirectionalForStream(conn, c)
	}))

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
