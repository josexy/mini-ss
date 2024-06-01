package proxyaddons

type reject struct{}

func (*reject) Request(ctx *Context) {
	ctx.Reject(errContextReject)
}

func (*reject) Response(ctx *Context) {
	ctx.Reject(errContextReject)
}

func (*reject) Message(ctx *Context) {
	ctx.Reject(errContextReject)
}
