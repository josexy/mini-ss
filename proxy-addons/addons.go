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

var (
	requestProxyAddonsList  []RequestHandler
	responseProxyAddonsList []ResponseHandler
	messageProxyAddonsList  []MessageHandler
)

var (
	MutableHTTPInterceptor proxy.MutableHTTPInterceptor
	MutableWSInterceptor   proxy.MutableWebsocketInterceptor
)

var uniqueId uint64

func increaseId() uint64 { return atomic.AddUint64(&uniqueId, 1) }

type (
	ProxyAddons     interface{}
	RequestHandler  interface{ Request(*Context) }
	ResponseHandler interface{ Response(*Context) }
	MessageHandler  interface{ Message(*Context) }
)

func init() {

	Use(&duration{}, &logEvent{}, &modifiedHeader{}, &dumper{})

	MutableHTTPInterceptor = func(req *http.Request, invoker proxy.HTTPDelegatedInvoker) (*http.Response, error) {
		var err error

		flow := &proxy.Flow{
			FlowID:    increaseId(),
			Timestamp: time.Now().UnixMilli(),
			HTTP: &proxy.HTTPFlow{
				Request: req,
			},
		}

		ctx := &Context{
			Flow:    flow,
			ctxT:    requestCtx,
			request: contextT[RequestHandler]{index: -1, chains: requestProxyAddonsList},
		}

		ctx.Next()
		if err = ctx.error(); err != nil {
			return nil, err
		}

		// deep copy http request body content
		cloneReq, err := deepCopyRequest(flow.HTTP.Request)
		if err != nil {
			return nil, err
		}

		rsp, err := invoker.Invoke(flow.HTTP.Request)
		if err != nil {
			return nil, err
		}
		flow.HTTP.Request = cloneReq
		flow.HTTP.Response = rsp

		ctx.ctxT = responseCtx
		ctx.response = contextT[ResponseHandler]{index: -1, chains: responseProxyAddonsList}
		ctx.Next()
		if err = ctx.error(); err != nil {
			return nil, err
		}
		return rsp, nil
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

		ctx := &Context{
			Flow:    flow,
			ctxT:    messageCtx,
			message: contextT[MessageHandler]{index: -1, chains: messageProxyAddonsList},
		}

		ctx.Next()
		if err = ctx.error(); err != nil {
			return err
		}

		if err = invoker.Invoke(flow.WS.MsgType, flow.WS.FramedData); err != nil {
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
