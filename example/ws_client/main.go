package main

import (
	"fmt"
	"log"

	"github.com/josexy/mini-ss/transport"
)

func main() {
	dialer := transport.NewDialer(transport.Websocket, transport.DefaultWsOptions)
	conn, err := dialer.Dial("127.0.0.1:10086")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		fmt.Printf("--> %s, %d\n", buf[:n], n)
	}
}
