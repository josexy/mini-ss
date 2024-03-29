package main

import (
	"io"
	"log"
	"os"
	"sync"

	"github.com/josexy/mini-ss/transport"
)

func main() {
	dialer := transport.NewDialer(transport.Grpc, transport.DefaultGrpcOptions)
	conn, err := dialer.Dial("127.0.0.1:10086")
	if err != nil {
		log.Fatalln(err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	fn := func(dest io.WriteCloser, src io.Reader) {
		defer wg.Done()
		_, _ = io.Copy(dest, src)
		_ = dest.Close()
	}
	// GET / HTTP/1.1
	go fn(conn, os.Stdin)
	go fn(os.Stdout, conn)
	wg.Wait()
}
