package proxyaddons

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/josexy/mini-ss/proxy"
)

var addonsList []ProxyAddons

var (
	MutableHTTPInterceptor proxy.MutableHTTPInterceptor
	MutableWSInterceptor   proxy.MutableWebsocketInterceptor
)

var uniqueId uint64

func increaseId() uint64 { return atomic.AddUint64(&uniqueId, 1) }

type ProxyAddons interface {
	Name() string
	Init()
}

func executeAddons(fn func(addons ProxyAddons) error) (err error) {
	for _, addons := range addonsList {
		if fn != nil {
			if err = fn(addons); err != nil {
				break
			}
		}
	}
	return
}

func init() {

	addonsList = []ProxyAddons{
		// &logEvent{},
		&dumper{},
	}

	executeAddons(func(addons ProxyAddons) error {
		addons.Init()
		return nil
	})

	MutableHTTPInterceptor = func(req *http.Request, invoker proxy.HTTPDelegatedInvoker) (*http.Response, error) {
		var err error

		flow := &proxy.Flow{
			FlowID:    increaseId(),
			Timestamp: time.Now().UnixMilli(),
			HTTP: &proxy.HTTPFlow{
				Request: req,
			},
		}
		if err = executeAddons(func(addons ProxyAddons) error {
			if handler, ok := addons.(proxy.HTTPFlowHandler); ok {
				return handler.Request(flow)
			}
			return nil
		}); err != nil {
			return nil, err
		}

		// deep copy http request body content
		cloneReq, err := deepCopyRequest(flow.HTTP.Request)
		if err != nil {
			return nil, err
		}

		rsp, err := invoker.Invoke(flow.HTTP.Request)
		if err != nil {
			executeAddons(func(addons ProxyAddons) error {
				if handler, ok := addons.(proxy.ErrorFlowHandler); ok {
					handler.Error(err)
				}
				return nil
			})
			return nil, err
		}
		flow.HTTP.Request = cloneReq
		flow.HTTP.Response = rsp
		if err = executeAddons(func(addons ProxyAddons) error {
			if handler, ok := addons.(proxy.HTTPFlowHandler); ok {
				return handler.Response(flow)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		return rsp, err
	}
	MutableWSInterceptor = func(dir proxy.WSDirection, req *http.Request, msgType int, data []byte, invoker proxy.WebsocketDelegatedInvoker) error {
		var err error
		flow := &proxy.Flow{
			FlowID:    increaseId(),
			Timestamp: time.Now().UnixMilli(),
			WS: &proxy.WSFlow{
				Direction:  dir,
				Request:    req,
				MsgType:    msgType,
				FramedData: data,
			},
		}
		if err = executeAddons(func(addons ProxyAddons) error {
			if handler, ok := addons.(proxy.WSFlowHandler); ok {
				return handler.Message(flow)
			}
			return nil
		}); err != nil {
			return err
		}

		if err = invoker.Invoke(flow.WS.MsgType, flow.WS.FramedData); err != nil {
			executeAddons(func(addons ProxyAddons) error {
				if handler, ok := addons.(proxy.ErrorFlowHandler); ok {
					handler.Error(err)
				}
				return nil
			})
			return err
		}
		return nil
	}
}

func deepCopyRequest(req *http.Request) (*http.Request, error) {
	cloneReq := req.Clone(context.Background())
	if req.Body == nil {
		return cloneReq, nil
	}
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body.Close()
	req.Body = io.NopCloser(&buf)
	cloneReq.Body = io.NopCloser(bytes.NewBuffer(buf.Bytes()))
	return cloneReq, nil
}
