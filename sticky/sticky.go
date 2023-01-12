package sticky

import (
	"io"
	"net"
	"sync"
)

type SharedReader struct {
	r io.Reader
	net.Conn
	mu sync.Mutex
}

func NewSharedReader(r io.Reader, conn net.Conn) *SharedReader {
	return &SharedReader{
		r:    r,
		Conn: conn,
	}
}

func (c *SharedReader) Read(b []byte) (int, error) {
	c.mu.Lock()
	if c.r == nil {
		c.mu.Unlock()
		return c.Conn.Read(b)
	}
	n, err := c.r.Read(b)
	if err == io.EOF {
		var n2 int
		n2, err = c.Conn.Read(b[n:])
		n += n2
	}
	c.mu.Unlock()
	return n, err
}
