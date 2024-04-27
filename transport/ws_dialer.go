package transport

import (
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/mini-ss/connection"
)

// WsProxyFuncForTesting this global function used for testing
var WsProxyFuncForTesting func(req *http.Request) (*url.URL, error)

type wsDialer struct {
	tcpDialer
	Opts *WsOptions
}

func (d *wsDialer) Dial(addr string) (net.Conn, error) {
	scheme := "ws"
	tlsConfig, err := d.Opts.GetClientTlsConfig()
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		scheme = "wss"
	}
	urls := url.URL{
		Scheme: scheme,
		Host:   addr,
		Path:   d.Opts.Path,
	}
	dialer := &websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return d.tcpDialer.Dial(addr)
		},
		Proxy:             WsProxyFuncForTesting,
		ReadBufferSize:    d.Opts.RevBuffer,
		WriteBufferSize:   d.Opts.SndBuffer,
		EnableCompression: d.Opts.Compress,
		TLSClientConfig:   tlsConfig,
		HandshakeTimeout:  30 * time.Second,
	}
	header := http.Header{}
	header.Set("Host", d.Opts.Host)
	if d.Opts.UserAgent != "" {
		header.Add("User-Agent", d.Opts.UserAgent)
	}
	conn, resp, err := dialer.Dial(urls.String(), header)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	return connection.NewWebsocketConn(conn), nil
}
