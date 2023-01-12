package main

import (
	"net"
	"os"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

type customSrv struct{}

func (customSrv) ServeKCP(conn net.Conn) {
	logx.Info("%s", conn.RemoteAddr().String())
	tcpConn, err := transport.DialTCP("127.0.0.1:10002")
	if err != nil {
		logx.ErrorBy(err)
		return
	}
	defer tcpConn.Close()
	transport.RelayTCP(conn, tcpConn)
}

func main() {
	srv := server.NewKcpServer(":10001", &customSrv{}, transport.DefaultKcpOptions)

	interrupt := make(chan os.Signal, 1)
	go srv.Start()

	if err := <-srv.Error(); err != nil {
		logx.FatalBy(err)
	}

	<-interrupt
	srv.Close()
}
