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
	"github.com/josexy/mini-ss/server"
)

func main() {
	options := &options.WsOptions{
		Host:      "www.baidu.com",
		Path:      "/ws",
		SndBuffer: 4096,
		RevBuffer: 4096,
		Compress:  false,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36",
		// TlsOptions: options.TlsOptions{
		// 	Mode:     options.TLS,
		// 	KeyFile:  "certs/server.key",
		// 	CertFile: "certs/server.crt",
		// },
	}
	srv := server.NewWsServer(":8080", nil, options)
	srv.Handler = server.WsHandlerFunc(func(c net.Conn) {
		log.Println(c.LocalAddr(), c.RemoteAddr())
		buf := make([]byte, 1024)
		for {
			n, err := c.Read(buf)
			if err != nil {
				log.Println(err)
				break
			}
			_, err = c.Write([]byte("--> " + string(buf[:n])))
			if err != nil {
				log.Println(err)
				return
			}
		}
	})
	go func() {
		err := srv.Start(context.Background())
		log.Println("close server with err:", err)
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	<-interrupt

	srv.Close()
	time.Sleep(time.Second)
}
