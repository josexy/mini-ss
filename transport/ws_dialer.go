package transport

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/options"
)

// WsProxyFuncForTesting this global function used for testing
var WsProxyFuncForTesting func(req *http.Request) (*url.URL, error)

type wsDialer struct {
	tcpDialer
	opts *options.WsOptions
}

func (d *wsDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	scheme := "ws"
	tlsConfig, err := d.opts.GetClientTlsConfig()
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		scheme = "wss"
	}
	urls := url.URL{
		Scheme: scheme,
		Host:   addr,
		Path:   d.opts.Path,
	}
	dialer := &websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return d.tcpDialer.Dial(ctx, addr)
		},
		Proxy:             WsProxyFuncForTesting,
		ReadBufferSize:    d.opts.RevBuffer,
		WriteBufferSize:   d.opts.SndBuffer,
		EnableCompression: d.opts.Compress,
		TLSClientConfig:   tlsConfig,
		HandshakeTimeout:  30 * time.Second,
	}
	header := http.Header{}
	header.Set("Host", d.opts.Host)
	if d.opts.UserAgent != "" {
		header.Add("User-Agent", d.opts.UserAgent)
	}
	conn, resp, err := dialer.Dial(urls.String(), header)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return connection.NewWebsocketConn(conn), nil
}
