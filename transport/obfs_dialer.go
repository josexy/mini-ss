package transport

import (
	"context"
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/options"
)

type obfsDialer struct {
	tcpDialer
	opts *options.ObfsOptions
}

func (d *obfsDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	conn, err := d.tcpDialer.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return connection.NewObfsConn(conn, d.opts.Host, false), nil
}
