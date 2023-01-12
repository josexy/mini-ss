package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/ss"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	ruler := dns.NewRuler(dns.Global, "", "", nil)
	geoip.Data, _ = os.ReadFile("Country.mmdb")

	srv := ss.NewShadowsocksClient(
		ss.WithServerCompose(
			ss.WithServerAddr("127.0.0.1:8388"),
			ss.WithMethod("none"),
			ss.WithPassword("12345"),
		),
		ss.WithSocksAddr("127.0.0.1:10086"),
		ss.WithHttpAddr("127.0.0.1:10087"),
		ss.WithRuler(ruler),
	)

	done := make(chan struct{})
	go func() {
		if err := srv.Start(); err != nil && err != server.ErrServerClosed {
			logx.ErrorBy(err)
		}
		done <- struct{}{}
	}()

	<-interrupt

	srv.Close()
}
