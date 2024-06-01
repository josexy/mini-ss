package proxyaddons

import (
	"time"

	"github.com/josexy/mini-ss/util/logger"
)

type duration struct{}

func (*duration) Request(ctx *Context) {
	start := time.Now()
	ctx.Next()
	end := time.Now()
	logger.Logger.Debugf("addons[duration]: request: %s", end.Sub(start))
}

func (*duration) Response(ctx *Context) {
	start := time.Now()
	ctx.Next()
	end := time.Now()
	logger.Logger.Debugf("addons[duration]: response: %s", end.Sub(start))
}

func (*duration) Message(ctx *Context) {
	start := time.Now()
	ctx.Next()
	end := time.Now()
	logger.Logger.Debugf("addons[duration]: message: %s", end.Sub(start))
}
