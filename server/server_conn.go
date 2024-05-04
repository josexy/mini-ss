package server

import (
	"errors"
	"net"
	"runtime"
	"sync/atomic"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/logger"
)

var errClosed = errors.New("connection closed")

type onceCloseConn struct {
	net.Conn
	closed uint32
}

func (c *onceCloseConn) Close() error {
	if atomic.LoadUint32(&c.closed) != 0 {
		return errClosed
	}
	atomic.StoreUint32(&c.closed, 1)
	return c.Conn.Close()
}

type Conn struct {
	*onceCloseConn
	server Server
}

func newConn(conn net.Conn, server Server) *Conn {
	return &Conn{
		server:        server,
		onceCloseConn: &onceCloseConn{Conn: conn},
	}
}

func (c *Conn) serve() {
	defer func() {
		// if an error occurs, close the client connection
		_ = c.onceCloseConn.Close()
		if err := recover(); err != nil {
			buf := stackTraceBufferPool.Get()
			n := runtime.Stack(*buf, false)
			logger.Logger.Error("connection recovery",
				logx.Error("err", err.(error)),
				logx.String("stackbuf", string((*buf)[:n])),
			)
			stackTraceBufferPool.Put(buf)
		}
	}()
	c.server.Serve(c)
}
