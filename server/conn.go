package server

import (
	"net"
	"runtime"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/statistic"
)

type Conn struct {
	conn       net.Conn
	packetConn net.PacketConn
	server     Server
	isPacket   bool
}

func newConn(conn net.Conn, packetConn net.PacketConn, server Server) Conn {
	if statistic.DefaultManager != nil {
		if conn != nil {
			// tcp tracker
			conn = statistic.NewTcpTracker(conn,
				conn.RemoteAddr().String(),
				conn.LocalAddr().String(),
				statistic.LazyContext{},
				statistic.DefaultManager)
		} else {
			// udp tracker
			packetConn = statistic.NewUdpTracker(packetConn,
				"",
				packetConn.LocalAddr().String(),
				statistic.LazyContext{},
				statistic.DefaultManager)
		}
	}
	c := Conn{
		conn:       conn,
		packetConn: packetConn,
		server:     server,
	}
	if packetConn != nil {
		c.isPacket = true
	}

	return c
}

func (c *Conn) close() error {
	if c.isPacket {
		return c.packetConn.Close()
	}
	return c.conn.Close()
}

func (c *Conn) serve() {
	if !c.isPacket {
		defer func() {
			if err := recover(); err != nil {
				buf := stackTraceBufferPool.Get()
				n := runtime.Stack(*buf, false)
				logx.Error("%v\n%s", err, (*buf)[:n])
				stackTraceBufferPool.Put(buf)
			}
		}()
	}
	c.server.Serve(c)
}
