package obfs

import "net"

type plain struct{}

func newPlain(b *Base) Obfs {
	return &plain{}
}

func (p *plain) StreamConn(c net.Conn) net.Conn { return c }
