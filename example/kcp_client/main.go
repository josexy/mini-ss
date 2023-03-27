package main

import (
	"log"
	"net"
	"os"

	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

type echoSrv struct{}

// multiplexing kcp dialer
var dialer = transport.NewDialer(transport.Kcp, transport.DefaultKcpOptions)

func (echoSrv) ServeTCP(conn net.Conn) {
	log.Println(conn.RemoteAddr().String())
	kcpConn, err := dialer.Dial("127.0.0.1:10001")
	if err != nil {
		log.Println(err)
		return
	}
	// the Close() don't close the raw kcp connection, it only closes the smux.Stream()
	defer kcpConn.Close()
	relay.RelayTCP(conn, kcpConn)
}

func main() {
	srv := server.NewTcpServer(":10000", &echoSrv{}, server.Tcp)

	interrupt := make(chan os.Signal, 1)
	go srv.Start()

	if err := <-srv.Error(); err != nil {
		panic(err)
	}

	<-interrupt
	srv.Close()
}
