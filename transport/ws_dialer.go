package transport

import (
	"context"
	"crypto/tls"
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
	err       error
	tlsConfig *tls.Config
	opts      *options.WsOptions
	dialer    *websocket.Dialer
}

func newWSDialer(opt options.Options) *wsDialer {
	opt.Update()
	wsOpts := opt.(*options.WsOptions)
	tlsConfig, err := wsOpts.GetClientTlsConfig()
	wsDialer := &wsDialer{
		err:       err,
		opts:      wsOpts,
		tlsConfig: tlsConfig,
	}
	dialer := &websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return wsDialer.tcpDialer.Dial(ctx, addr)
		},
		Proxy:             WsProxyFuncForTesting,
		ReadBufferSize:    wsOpts.RevBuffer,
		WriteBufferSize:   wsOpts.SndBuffer,
		EnableCompression: wsOpts.Compress,
		HandshakeTimeout:  30 * time.Second,
		TLSClientConfig:   tlsConfig,
	}
	wsDialer.dialer = dialer
	return wsDialer
}

func (d *wsDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	if d.err != nil {
		return nil, d.err
	}
	scheme := "ws"
	if d.tlsConfig != nil {
		scheme = "wss"
	}
	urls := url.URL{
		Scheme: scheme,
		Host:   addr,
		Path:   d.opts.Path,
	}
	header := http.Header{}
	header.Set("Host", d.opts.Host)
	if d.opts.UserAgent != "" {
		header.Add("User-Agent", d.opts.UserAgent)
	}
	conn, rsp, err := d.dialer.Dial(urls.String(), header)
	if err != nil {
		return nil, err
	}
	rsp.Body.Close()
	return connection.NewWebsocketConn(conn), nil
}
