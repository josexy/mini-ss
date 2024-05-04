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

type customSrv struct{}

func (customSrv) ServeQUIC(conn net.Conn) {
	log.Println(conn.RemoteAddr().String())
	tcpConn, err := transport.DialTCP(context.Background(), "127.0.0.1:10002")
	if err != nil {
		log.Println(err)
		return
	}
	defer tcpConn.Close()
	relay.IoCopyBidirectionalForStream(conn, tcpConn)
}

func main() {
	srv := server.NewQuicServer(":10001", &customSrv{}, transport.DefaultQuicOptions)

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
