package proxyaddons

import (
	"strconv"
)

type modifiedHeader struct{}

func (*modifiedHeader) Request(ctx *Context) {
	flow := ctx.Flow
	req := flow.HTTP.Request
	req.Header.Set("X-Request", strconv.Itoa(int(flow.FlowID)))
}

func (*modifiedHeader) Response(ctx *Context) {
	flow := ctx.Flow
	req := flow.HTTP.Response
	req.Header.Set("X-Response", strconv.Itoa(int(flow.FlowID)))
}
