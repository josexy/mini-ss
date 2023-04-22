package main

import (
	"log"
	"net"

	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	s := server.NewGrpcServer(":10086", nil, transport.DefaultGrpcOptions)
	s.Handler = server.GrpcHandler(server.GrpcHandlerFunc(func(c net.Conn) {
		log.Println(c.LocalAddr(), c.RemoteAddr())
		conn, err := net.Dial("tcp", "www.baidu.com:80")
		if err != nil {
			log.Println(err)
			return
		}
		relay.RelayTCP(conn, c)
	}))
	defer s.Close()
	s.Start()
}
