package proxyaddons

import (
	"fmt"
	"strings"
)

type dumper struct{}

func (*dumper) Request(ctx *Context) {}

func filter(contentType string) bool {
	// https://www.iana.org/assignments/media-types/media-types.xhtml#image
	ts := []string{
		"image",
		"font",
		"video",
		"audio",
	}
	for _, t := range ts {
		if strings.HasPrefix(contentType, t) {
			return true
		}
	}
	return false
}

func (*dumper) Response(ctx *Context) {
	flow := ctx.Flow
	contentType := flow.HTTP.Response.Header.Get("Content-Type")
	if filter(contentType) {
		return
	}
	view, err := flow.HTTP.DumpHTTPView()
	if err != nil {
		return
	}
	fmt.Printf("--------REQ---------\n%s\n--------------------\n", view.Request.Encode())
	fmt.Printf("--------RSP---------\n%s\n--------------------\n", view.Response.Encode())
}

func (*dumper) Message(ctx *Context) {
	fmt.Printf("--------MSG---------\n%s\n--------------------\n", ctx.Flow.WS.FramedData)
}
