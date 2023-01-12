package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/socks"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "socks":
			socksMain()
		case "tun":
			tcpTunMain()
		case "http":
			httpMain()
		case "echo":
			echoMain()
		}
	} else {
		wg := sync.WaitGroup{}
		wg.Add(15)
		for i := 0; i < 5; i++ {
			go func() {
				tcpTunMain()
			}()
		}
		for i := 0; i < 5; i++ {
			go func() {
				socksMain()
			}()
		}
		for i := 0; i < 5; i++ {
			go func() {
				httpMain()
			}()
		}
		wg.Wait()
	}
}

func tcpTunMain() {
	conn, err := transport.DialTCP(os.Args[2])
	if err != nil {
		logx.FatalBy(err)
	}
	defer conn.Close()
	for i := 0; i < 10; i++ {
		n, err := conn.Write([]byte("hello tcp server\n"))
		logx.Debug("sent bytes: %d, err: %v", n, err)
	}
}

func socksMain() {
	proxyCli := socks.NewSocks5Client("127.0.0.1:10086")
	proxyCli.SetSocksAuth("test", "12345678")
	defer proxyCli.Close()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return proxyCli.Dial(ctx, addr)
		},
	}
	cli := http.Client{
		Transport: transport,
		Timeout:   time.Second * 10,
	}
	fn := func(url string) {
		resp, err := cli.Get(url)
		if err != nil {
			logx.ErrorBy(err)
			return
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		logx.Debug("=> data [%s] len: %d\n", data[:20], len(data))
	}
	urls := []string{
		"https://www.baidu.com",
		"https://httpbin.org/get",
		"http://www.example.com",
		"http://ip.gs",
	}
	for _, url := range urls {
		for i := 0; i < 1; i++ {
			fn(url)
		}
	}
}

func httpMain() {
	fn := func(u string) {
		transport := &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				// return url.Parse("http://127.0.0.1:10087")
				return url.Parse("http://123:123@127.0.0.1:10087")
			},
		}
		cli := http.Client{
			Transport: transport,
			Timeout:   time.Second * 10,
		}
		resp, err := cli.Get(u)
		if err != nil {
			logx.ErrorBy(err)
			return
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		logx.Debug("url: %s => code [%s] len: %d\n", u, resp.StatusCode, len(data))
	}
	urls := []string{
		"http://httpbin.org/get",
		"https://www.baidu.com",
		// "http://127.0.0.1:8888",
	}
	for _, url := range urls {
		for i := 0; i < 3; i++ {
			fn(url)
		}
	}
}

func echoMain() {
	conn, err := transport.DialTCP("127.0.0.1:10000")
	if err != nil {
		logx.FatalBy(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() { io.Copy(os.Stdout, conn) }()
	go func() { io.Copy(conn, os.Stdin) }()
	wg.Wait()
}
