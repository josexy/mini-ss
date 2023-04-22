package transport

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/josexy/mini-ss/connection"
)

type wsDialer struct {
	tcpDialer
	Opts *WsOptions
}

func (d *wsDialer) Dial(addr string) (net.Conn, error) {
	scheme := "ws"
	var tlsConfig *tls.Config
	if d.Opts.TLS {
		scheme = "wss"
		tlsConfig = &tls.Config{InsecureSkipVerify: true}
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
		ReadBufferSize:    d.Opts.RevBuffer,
		WriteBufferSize:   d.Opts.SndBuffer,
		EnableCompression: d.Opts.Compress,
		HandshakeTimeout:  45 * time.Second,
		TLSClientConfig:   tlsConfig,
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
