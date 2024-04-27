package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
)

type WSDirection byte

const (
	Send WSDirection = iota
	Receive
)

type (
	HTTPDelegatedInvoker interface {
		Invoke(*http.Request) (*http.Response, error)
	}
	WebsocketDelegatedInvoker interface {
		Invoke(int, []byte) error
	}
	HTTPDelegatedInvokerFunc      func(*http.Request) (*http.Response, error)
	WebsocketDelegatedInvokerFunc func(int, []byte) error

	MutableHTTPInterceptor        func(*http.Request, HTTPDelegatedInvoker) (*http.Response, error)
	MutableWebsocketInterceptor   func(WSDirection, *http.Request, int, []byte, WebsocketDelegatedInvoker) error
	ImmutableHTTPInterceptor      func(*http.Request, *http.Response)
	ImmutableWebsocketInterceptor func(WSDirection, *http.Request, int, []byte)
)

func (f HTTPDelegatedInvokerFunc) Invoke(r *http.Request) (*http.Response, error) { return f(r) }
func (f WebsocketDelegatedInvokerFunc) Invoke(t int, data []byte) error           { return f(t, data) }

type baseImmutableHTTPInterceptor struct{ fn ImmutableHTTPInterceptor }

func (b baseImmutableHTTPInterceptor) InvokeInterceptor(req *http.Request, invoker HTTPDelegatedInvoker) (*http.Response, error) {
	copiedReq := cloneHttpRequest(req)
	rsp, err := invoker.Invoke(req)
	if err != nil {
		return rsp, err
	}
	copiedRsp := cloneHttpResponse(rsp)
	copiedRsp.Request = copiedReq
	b.fn(copiedReq, copiedRsp)
	return rsp, err
}

type baseMutableHTTPInterceptor struct{ fn MutableHTTPInterceptor }

func (b baseMutableHTTPInterceptor) InvokeInterceptor(req *http.Request, invoker HTTPDelegatedInvoker) (*http.Response, error) {
	return b.fn(req, invoker)
}

type baseImmutableWebsocketInterceptor struct{ fn ImmutableWebsocketInterceptor }

func (b baseImmutableWebsocketInterceptor) InvokeInterceptor(d WSDirection, req *http.Request, t int, data []byte, invoker WebsocketDelegatedInvoker) error {
	copiedData := make([]byte, len(data))
	copy(copiedData, data)
	err := invoker.Invoke(t, data)
	b.fn(d, req, t, copiedData)
	return err
}

type baseMutableWebsocketInterceptor struct{ fn MutableWebsocketInterceptor }

func (b baseMutableWebsocketInterceptor) InvokeInterceptor(d WSDirection, req *http.Request, t int, data []byte, invoker WebsocketDelegatedInvoker) error {
	return b.fn(d, req, t, data, invoker)
}

func copyBody(body io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	if body == nil {
		return nil, nil
	}
	rawData, err := io.ReadAll(body)
	if err != nil {
		return nil, nil
	}
	body.Close()
	return io.NopCloser(bytes.NewReader(rawData)), io.NopCloser(bytes.NewReader(rawData))
}

func cloneHttpRequest(req *http.Request) *http.Request {
	newReq := req.Clone(context.Background())
	newReq.Body, req.Body = copyBody(req.Body)
	return newReq
}

func cloneHttpResponse(rsp *http.Response) *http.Response {
	newRsp := new(http.Response)
	*newRsp = *rsp
	newRsp.Header = rsp.Header.Clone()
	newRsp.Trailer = rsp.Trailer.Clone()
	newRsp.Body, rsp.Body = copyBody(rsp.Body)
	newRsp.Request = nil
	if s := rsp.TransferEncoding; s != nil {
		s2 := make([]string, len(s))
		copy(s2, s)
		newRsp.TransferEncoding = s2
	}
	if cs := rsp.TLS; cs != nil {
		newRsp.TLS = new(tls.ConnectionState)
		*newRsp.TLS = *cs
	}
	return newRsp
}
