package connection

import (
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketConn struct {
	conn *websocket.Conn
	rbuf []byte // remaining buffer data
}

func NewWebsocketConn(c *websocket.Conn) *WebsocketConn { return &WebsocketConn{conn: c} }

func (c *WebsocketConn) Read(b []byte) (int, error) {
	var err error
	if len(c.rbuf) == 0 {
		_, c.rbuf, err = c.conn.ReadMessage()
	}
	n := copy(b, c.rbuf)
	c.rbuf = c.rbuf[n:]
	return n, err
}

func (c *WebsocketConn) Write(b []byte) (int, error) {
	return len(b), c.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (c *WebsocketConn) Close() error { return c.conn.Close() }

func (c *WebsocketConn) LocalAddr() net.Addr { return c.conn.LocalAddr() }

func (c *WebsocketConn) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }

func (c *WebsocketConn) SetReadDeadline(t time.Time) error { return c.conn.SetReadDeadline(t) }

func (c *WebsocketConn) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }

func (c *WebsocketConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}
