package connection

import (
	"net"

	"github.com/golang/snappy"
)

type CompressConn struct {
	net.Conn
	r *snappy.Reader
	w *snappy.Writer
}

func NewCompressConn(c net.Conn) *CompressConn {
	return &CompressConn{
		Conn: c,
		r:    snappy.NewReader(c),
		w:    snappy.NewBufferedWriter(c),
	}
}

func (c *CompressConn) Write(b []byte) (int, error) { return c.w.Write(b) }

func (c *CompressConn) Read(b []byte) (int, error) { return c.r.Read(b) }
