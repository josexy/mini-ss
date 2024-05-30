package proxyaddons

import (
	"errors"
	"net/http"

	"github.com/josexy/mini-ss/util/logger"
)

type reject struct{}

func (da *reject) Request(ctx *Context) {
	logger.Logger.Debug("addons[reject]: request")
	if ctx.HTTP.Request.Method == http.MethodGet {
		ctx.Reject(errors.New("reject request GET method"))
	}
}

func (da *reject) Response(ctx *Context) {
	logger.Logger.Debug("addons[reject]: response")
	if ctx.HTTP.Response.StatusCode == 200 {
		ctx.Reject(errors.New("reject response status code 200"))
	}
}

func (da *reject) Message(ctx *Context) {
	ctx.Reject(errors.New("reject message"))
}
