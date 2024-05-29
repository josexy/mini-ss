package proxyaddons

import (
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/proxy"
	"github.com/josexy/mini-ss/util/logger"
)

type logEvent struct{}

func (da *logEvent) Name() string { return "log_event" }

func (da *logEvent) Init() {}

func (da *logEvent) Request(*proxy.Flow) error { return nil }

func (da *logEvent) Response(flow *proxy.Flow) error {
	req := flow.HTTP.Request
	rsp := flow.HTTP.Response
	logger.Logger.Debug("http interceptor",
		logx.UInt64("id", flow.FlowID),
		logx.Time("timestamp", time.UnixMilli(flow.Timestamp)),
		logx.String("method", req.Method),
		logx.String("host", req.Host),
		logx.String("path", req.URL.Path),
		logx.String("content-type", req.Header.Get("Content-Type")),
		logx.Int("status", rsp.StatusCode),
		logx.Int64("size", rsp.ContentLength),
	)
	return nil
}

func (da *logEvent) Message(flow *proxy.Flow) error {
	req := flow.WS.Request
	direction := "Send"
	if flow.WS.Direction == proxy.Receive {
		direction = "Receive"
	}
	logger.Logger.Debug("ws interceptor",
		logx.UInt64("id", flow.FlowID),
		logx.Time("timestamp", time.UnixMilli(flow.Timestamp)),
		logx.String("direction", direction),
		logx.String("host", req.Host),
		logx.String("path", req.URL.Path),
		logx.Int("msg-type", flow.WS.MsgType),
		logx.Int("data-size", len(flow.WS.FramedData)),
		logx.String("data", string(flow.WS.FramedData)),
	)
	return nil
}
