package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	ruler := rule.NewRuler(rule.Global, "", "ws", nil)
	if err := geoip.OpenDB("Country.mmdb"); err != nil {
		panic(err)
	}

	srv := ss.NewShadowsocksClient(
		ss.WithServerCompose(
			ss.WithServerName("default"),
			ss.WithServerAddr("127.0.0.1:8388"),
			ss.WithMethod("xchacha20-ietf-poly1305"),
			ss.WithPassword("12345"),
			ss.WithUDPRelay(true),
		),
		ss.WithServerCompose(
			ss.WithServerName("ws"),
			ss.WithServerAddr("127.0.0.1:8389"),
			ss.WithMethod("chacha20-ietf-poly1305"),
			ss.WithPassword("12345"),
			ss.WithWsTransport(),
			ss.WithWsHost("www.baidu.com"),
			ss.WithWsTLS(transport.TLS),
			ss.WithWsPath("/ws"),
			ss.WithWsHostname("www.helloworld.com"),
			ss.WithWsCAPath("certs/ca.crt"),
			ss.WithUDPRelay(true),
		),
		ss.WithServerCompose(
			ss.WithServerName("grpc"),
			ss.WithServerAddr("127.0.0.1:8390"),
			ss.WithMethod("none"),
			ss.WithPassword("12345"),
			ss.WithGrpcTransport(),
			ss.WithGrpcTLS(transport.TLS),
			ss.WithGrpcHostname("www.helloworld.com"),
			ss.WithGrpcCAPath("certs/ca.crt"),
			ss.WithUDPRelay(true),
		),
		ss.WithSocksAddr("127.0.0.1:10086"),
		ss.WithHttpAddr("127.0.0.1:10087"),
		ss.WithRuler(ruler),
	)

	go func() {
		if err := srv.Start(); err != nil {
			panic(err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT)
	<-interrupt

	srv.Close()
	time.Sleep(time.Second * 2)
}
