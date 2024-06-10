package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
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
		time.Sleep(time.Millisecond * 20)
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
		log.Println(buf.Len())
	}

	for i := 0; i < 10; i++ {
		go request()
		time.Sleep(time.Millisecond * 20)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	<-interrupt
	time.Sleep(time.Millisecond * 200)
}
