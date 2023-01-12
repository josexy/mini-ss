package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/resolver"
)

func main() {
	srv := dns.NewDnsServer(":53")
	resolver.DefaultResolver = resolver.NewDnsResolver(nil)
	resolver.DefaultResolver.EnableFakeIP("10.10.0.1/16")

	inter := make(chan os.Signal, 1)
	signal.Notify(inter, syscall.SIGINT)
	go func() {
		srv.Start()
	}()
	logx.Debug("start dns server: %s", srv.LocalAddress())
	<-inter
	srv.Close()
}
