package connection

import (
	"bufio"
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

type BufioConn struct {
	net.Conn
	r *bufio.Reader
}

func NewBufioConn(c net.Conn) *BufioConn { return &BufioConn{Conn: c, r: bufio.NewReader(c)} }

func (c *BufioConn) Peek(n int) ([]byte, error) { return c.r.Peek(n) }

func (c *BufioConn) Read(p []byte) (int, error) { return c.r.Read(p) }
