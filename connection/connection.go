package connection

import (
	"io"
	"net"
)

type ConnWithReader struct {
	net.Conn
	reader io.Reader
}

func NewConnWithReader(conn net.Conn, readers ...io.Reader) *ConnWithReader {
	var reader io.Reader
	if len(readers) > 0 {
		reader = io.MultiReader(append(readers, conn)...)
	}
	return &ConnWithReader{
		Conn:   conn,
		reader: reader,
	}
}

func (c *ConnWithReader) Read(b []byte) (int, error) {
	if c.reader == nil {
		return c.Conn.Read(b)
	}
	return c.reader.Read(b)
}
