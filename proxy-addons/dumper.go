package proxyaddons

import (
	"fmt"
	"strings"

	"github.com/josexy/mini-ss/util/logger"
)

type dumper struct{}

func (da *dumper) Request(ctx *Context) {
	logger.Logger.Debug("addons[dumper]: request")
}

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

func (da *dumper) Response(ctx *Context) {
	logger.Logger.Debug("addons[dumper]: response")
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

func (da *dumper) Message(ctx *Context) {
	fmt.Printf("--------MSG---------\n%s\n--------------------\n", ctx.Flow.WS.FramedData)
}
