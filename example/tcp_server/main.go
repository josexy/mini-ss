package main

import (
	"bytes"
	"net"
	"os"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/server"
)

func main() {
	echoMain()
}

type echoSrv struct{}

func (echoSrv) ServeTCP(conn net.Conn) {
	logx.Info("%s", conn.RemoteAddr().String())
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			logx.ErrorBy(err)
			break
		}
		xbuf := buf[:n]
		logx.Info("-> %d %s", n, string(xbuf))
		conn.Write(append(bytes.TrimRight(xbuf, "\r\n"), " <-- echo from server\n"...))
	}
}

// tcp client <-> kcp/quic client <-> kcp/quic server <-> tcp server

func echoMain() {
	srv := server.NewTcpServer(":10002", &echoSrv{}, server.Tcp)

	interrupt := make(chan os.Signal, 1)
	go srv.Start()

	if err := <-srv.Error(); err != nil {
		logx.FatalBy(err)
	}

	<-interrupt
	srv.Close()
}
