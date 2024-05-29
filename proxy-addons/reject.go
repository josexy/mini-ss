package proxyaddons

import (
	"fmt"

	"github.com/josexy/mini-ss/proxy"
)

type reject struct{}

func (da *reject) Name() string { return "reject" }

func (da *reject) Init() {}

func (da *reject) Request(flow *proxy.Flow) error {
	return fmt.Errorf("reject http request: %s", flow.HTTP.Request.URL)
}

func (da *reject) Response(flow *proxy.Flow) error {
	return fmt.Errorf("reject http response: %s", flow.HTTP.Request.URL)
}

func (da *reject) Message(flow *proxy.Flow) error {
	return fmt.Errorf("reject ws message: %s", flow.WS.Request.URL)
}
