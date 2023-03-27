package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/ss"
)

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := ss.NewShadowsocksServer(
		ss.WithServerCompose(
			ss.WithServerAddr(":8388"),
			ss.WithMethod("none"),
			ss.WithPassword("12345"),
		),
	)

	done := make(chan struct{})
	go func() {
		if err := srv.Start(); err != nil && err != server.ErrServerClosed {
			panic(err)
		}
		done <- struct{}{}
	}()

	<-interrupt

	srv.Close()
}
