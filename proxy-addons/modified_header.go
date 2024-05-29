package proxyaddons

import (
	"strconv"

	"github.com/josexy/mini-ss/proxy"
)

type modifiedHeader struct{}

func (mh *modifiedHeader) Name() string { return "modified_header" }

func (mh *modifiedHeader) Init() {}

func (mh *modifiedHeader) Request(flow *proxy.Flow) error {
	req := flow.HTTP.Request
	req.Header.Set("X-Request", strconv.Itoa(int(flow.FlowID)))
	return nil
}

func (mh *modifiedHeader) Response(flow *proxy.Flow) error {
	req := flow.HTTP.Response
	req.Header.Set("X-Response", strconv.Itoa(int(flow.FlowID)))
	return nil
}
