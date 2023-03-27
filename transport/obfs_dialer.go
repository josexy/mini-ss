package transport

import "net"

type obfsDialer struct {
	tcpDialer
	Opts *ObfsOptions
}

func (d *obfsDialer) Dial(addr string) (net.Conn, error) {
	conn, err := d.tcpDialer.Dial(addr)
	if err != nil {
		return nil, err
	}
	return NewObfsConn(conn, d.Opts.Host, false), nil
}
