package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/josexy/mini-ss/transport"
)

func main() {
	// websocket via http or socks5 proxy
	transport.WsProxyFuncForTesting = func(req *http.Request) (*url.URL, error) {
		return url.Parse("socks5://127.0.0.1:10086")
	}
	options := &transport.WsOptions{
		Host:      "www.baidu.com",
		Path:      "/ws",
		SndBuffer: 4096,
		RevBuffer: 4096,
		Compress:  false,
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36",
		TlsOptions: transport.TlsOptions{
			Mode:     transport.TLS,
			Hostname: "127.0.0.1",
			CAFile:   "certs/ca.crt",
		},
	}
	dialer := transport.NewDialer(transport.Websocket, options)
	conn, err := dialer.Dial("127.0.0.1:8080")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	go func() {
		for i := 0; i < 10; i++ {
			conn.Write([]byte("hello world"))
		}
	}()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Println(err)
				break
			}
			fmt.Println(string(buf[:n]))
		}
	}()

	time.Sleep(time.Second)
}
