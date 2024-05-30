package proxyaddons

import (
	"time"

	"github.com/josexy/mini-ss/util/logger"
)

type duration struct{}

func (d *duration) Request(ctx *Context) {
	logger.Logger.Debug("addons[duration]: request")
	start := time.Now()
	ctx.Next()
	end := time.Now()
	logger.Logger.Debugf("addons[duration]: request: %s", end.Sub(start))
}

func (d *duration) Response(ctx *Context) {
	logger.Logger.Debug("addons[duration]: response")
	start := time.Now()
	ctx.Next()
	end := time.Now()
	logger.Logger.Debugf("addons[duration]: response: %s", end.Sub(start))
}
