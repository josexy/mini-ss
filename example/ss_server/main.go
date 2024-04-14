package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/transport"
)

func main() {
	srv := ss.NewShadowsocksServer(
		ss.WithServerCompose(
			ss.WithServerName("default"),
			ss.WithServerAddr(":8388"),
			ss.WithMethod("xchacha20-ietf-poly1305"),
			ss.WithPassword("12345"),
			ss.WithDefaultTransport(),
			ss.WithUDPRelay(true),
		),
		ss.WithServerCompose(
			ss.WithServerName("ws"),
			ss.WithServerAddr(":8389"),
			ss.WithMethod("chacha20-ietf-poly1305"),
			ss.WithPassword("12345"),
			ss.WithWsTransport(),
			ss.WithWsHost("www.baidu.com"),
			ss.WithWsTLS(transport.TLS),
			ss.WithWsPath("/ws"),
			ss.WithWsKeyPath("certs/server.key"),
			ss.WithWsCertPath("certs/server.crt"),
			ss.WithUDPRelay(true),
		),
		ss.WithServerCompose(
			ss.WithServerName("grpc"),
			ss.WithServerAddr(":8390"),
			ss.WithMethod("none"),
			ss.WithPassword("12345"),
			ss.WithGrpcTransport(),
			ss.WithGrpcTLS(transport.TLS),
			ss.WithGrpcKeyPath("certs/server.key"),
			ss.WithGrpcCertPath("certs/server.crt"),
			ss.WithUDPRelay(true),
		),
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
	time.Sleep(time.Second * 1)
}
