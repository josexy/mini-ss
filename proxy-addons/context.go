package proxyaddons

import (
	"errors"
	"math"

	"github.com/josexy/mini-ss/proxy"
)

var errContextReject = errors.New("context reject")

const abortIndex = math.MaxInt8 >> 1

const (
	requestCtx byte = iota + 1
	responseCtx
	messageCtx
)

type contextT[T any] struct {
	err    error
	index  int8
	chains []T
}

func (c *contextT[T]) next(fn func(T)) {
	c.index++
	for c.index < int8(len(c.chains)) {
		fn(c.chains[c.index])
		c.index++
	}
}

func (c *contextT[T]) abort() { c.index = abortIndex }

type Context struct {
	*proxy.Flow
	ctxT     byte
	request  contextT[RequestHandler]
	response contextT[ResponseHandler]
	message  contextT[MessageHandler]
}

func (c *Context) Next() {
	switch c.ctxT {
	case requestCtx:
		c.request.next(func(h RequestHandler) { h.Request(c) })
	case responseCtx:
		c.response.next(func(h ResponseHandler) { h.Response(c) })
	case messageCtx:
		c.message.next(func(h MessageHandler) { h.Message(c) })
	}
}

func (c *Context) Abort() {
	switch c.ctxT {
	case requestCtx:
		c.request.abort()
	case responseCtx:
		c.response.abort()
	case messageCtx:
		c.message.abort()
	}
}

func (c *Context) error() error {
	switch c.ctxT {
	case requestCtx:
		return c.request.err
	case responseCtx:
		return c.response.err
	case messageCtx:
		return c.message.err
	}
	return nil
}

func (c *Context) Reject(err error) {
	if err == nil {
		err = errContextReject
	}
	switch c.ctxT {
	case requestCtx:
		c.request.abort()
		c.request.err = err
	case responseCtx:
		c.response.abort()
		c.response.err = err
	case messageCtx:
		c.message.abort()
		c.message.err = err
	}
}

func Use(addons ...ProxyAddons) {
	for _, pa := range addons {
		if h, ok := pa.(RequestHandler); ok {
			requestProxyAddonsList = append(requestProxyAddonsList, h)
		}
		if h, ok := pa.(ResponseHandler); ok {
			responseProxyAddonsList = append(responseProxyAddonsList, h)
		}
		if h, ok := pa.(MessageHandler); ok {
			messageProxyAddonsList = append(messageProxyAddonsList, h)
		}
	}
}
