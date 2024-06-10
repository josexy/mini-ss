package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

type echoSrv struct{}

var dialer = transport.NewDialer(transport.Quic, &options.QuicOptions{
	Conns: 3,
	// TlsOptions: options.TlsOptions{
	// 	Mode:     options.TLS,
	// 	Hostname: "127.0.0.1",
	// 	CAFile:   "certs/ca.crt",
	// },
})

func (echoSrv) ServeTCP(conn net.Conn) {
	log.Println(conn.RemoteAddr().String())
	quicConn, err := dialer.Dial(context.Background(), "127.0.0.1:10001")
	if err != nil {
		log.Println(err)
		return
	}
	defer quicConn.Close()
	relay.IoCopyBidirectionalForStream(conn, quicConn)
}

func main() {
	srv := server.NewTcpServer(":10000", &echoSrv{}, server.Tcp)
	go func() {
		err := srv.Start(context.Background())
		log.Println("close server with err:", err)
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	<-interrupt

	srv.Close()
	time.Sleep(time.Millisecond * 200)
}
