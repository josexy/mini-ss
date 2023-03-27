package main

import (
	"log"
	"net"
	"os"

	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

type customSrv struct{}

func (customSrv) ServeKCP(conn net.Conn) {
	log.Println(conn.RemoteAddr().String())
	tcpConn, err := transport.DialTCP("127.0.0.1:10002")
	if err != nil {
		log.Println(err)
		return
	}
	defer tcpConn.Close()
	relay.RelayTCP(conn, tcpConn)
}

func main() {
	srv := server.NewKcpServer(":10001", &customSrv{}, transport.DefaultKcpOptions)

	interrupt := make(chan os.Signal, 1)
	go srv.Start()

	if err := <-srv.Error(); err != nil {
		panic(err)
	}

	<-interrupt
	srv.Close()
}
