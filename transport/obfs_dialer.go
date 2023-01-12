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
	// obfs-tls
	if d.Opts.TLS {
		return NewObfsTLSConn(conn, d.Opts.Host, false), nil
	}
	return NewObfsConn(conn, d.Opts.Host, false), nil
}
