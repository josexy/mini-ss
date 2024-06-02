package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	dialer := transport.NewDialer(transport.Ssh, &options.SshOptions{
		User:       "test",
		Password:   "test",
		PrivateKey: "ssh-keys/test-key",
		PublicKey:  "ssh-keys/test-key.pub",
	})

	request := func() {
		conn, err := dialer.Dial(context.Background(), "127.0.0.1:10086")
		if err != nil {
			log.Fatalln(err)
		}
		defer conn.Close()
		go func() {
			conn.Write([]byte("GET / HTTP/1.1\r\n\r\n\r\n"))
		}()
		buf := &bytes.Buffer{}
		buf.ReadFrom(conn)
		fmt.Println(buf.Len())
	}

	for i := 0; i < 2; i++ {
		go request()
	}

	time.Sleep(time.Second * 4)
}
