package proxyaddons

import (
	"strconv"

	"github.com/josexy/mini-ss/util/logger"
)

type modifiedHeader struct{}

func (mh *modifiedHeader) Request(ctx *Context) {
	logger.Logger.Debug("addons[modifiedHeader]: request")
	flow := ctx.Flow
	req := flow.HTTP.Request
	req.Header.Set("X-Request", strconv.Itoa(int(flow.FlowID)))
}

func (mh *modifiedHeader) Response(ctx *Context) {
	logger.Logger.Debug("addons[modifiedHeader]: response")
	flow := ctx.Flow
	req := flow.HTTP.Response
	req.Header.Set("X-Response", strconv.Itoa(int(flow.FlowID)))
}
