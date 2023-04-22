package transport

import (
	"net"

	"github.com/josexy/mini-ss/connection"
)

type obfsDialer struct {
	tcpDialer
	Opts *ObfsOptions
}

func (d *obfsDialer) Dial(addr string) (net.Conn, error) {
	conn, err := d.tcpDialer.Dial(addr)
	if err != nil {
		return nil, err
	}
	return connection.NewObfsConn(conn, d.Opts.Host, false), nil
}
