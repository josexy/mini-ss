package main

import (
	"net"
	"os"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

type echoSrv struct{}

// multiplexing quic dialer
var dialer = transport.NewDialer(transport.QUIC, nil)

func (echoSrv) ServeTCP(conn net.Conn) {
	logx.Info("%s", conn.RemoteAddr().String())

	quicConn, err := dialer.Dial("127.0.0.1:10001")
	if err != nil {
		logx.ErrorBy(err)
		return
	}
	// the Close() don't close the raw quic connection, it only closes the smux.Stream()
	defer quicConn.Close()
	transport.RelayTCP(conn, quicConn)
}

func main() {
	srv := server.NewTcpServer(":10000", &echoSrv{}, server.Tcp)

	interrupt := make(chan os.Signal, 1)
	go srv.Start()

	if err := <-srv.Error(); err != nil {
		logx.ErrorBy(err)
		return
	}

	<-interrupt
	srv.Close()
}
