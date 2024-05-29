package proxyaddons

import (
	"fmt"
	"strings"

	"github.com/josexy/mini-ss/proxy"
)

type dumper struct{}

func (da *dumper) Name() string { return "dumper" }

func (da *dumper) Init() {}

func (da *dumper) Request(flow *proxy.Flow) error { return nil }

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
func (da *dumper) Response(flow *proxy.Flow) error {
	contentType := flow.HTTP.Response.Header.Get("Content-Type")
	if filter(contentType) {
		return nil
	}
	view, err := flow.HTTP.DumpHTTPView()
	if err != nil {
		return err
	}
	fmt.Printf("--------REQ---------\n%s\n--------------------\n", view.Request.Encode())
	fmt.Printf("--------RSP---------\n%s\n--------------------\n", view.Response.Encode())
	return nil
}

func (da *dumper) Message(flow *proxy.Flow) error {
	fmt.Printf("--------MSG---------\n%s\n--------------------\n", flow.WS.FramedData)
	return nil
}
