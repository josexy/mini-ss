package mux

import (
	"net"

	smux "github.com/xtaci/smux"
)

type MuxStreamConn struct {
	net.Conn
	Stream *smux.Stream
}

func (c *MuxStreamConn) Read(b []byte) (int, error) {
	return c.Stream.Read(b)
}

func (c *MuxStreamConn) Write(b []byte) (int, error) {
	return c.Stream.Write(b)
}

// Close don't close the raw net.Conn connection, it just closes the smux.Stream
func (c *MuxStreamConn) Close() error {
	return c.Stream.Close()
}

// MuxBindSession bind net.Conn and smux.Session
type MuxBindSession struct {
	conn    net.Conn
	Session *smux.Session
}

func NewMuxBindSession(c net.Conn, sess *smux.Session) *MuxBindSession {
	return &MuxBindSession{
		conn:    c,
		Session: sess,
	}
}

// GetStreamConn multiplex connection
func (session *MuxBindSession) GetStreamConn() (net.Conn, error) {
	streamConn, err := session.Session.OpenStream()
	if err != nil {
		return nil, err
	}
	return &MuxStreamConn{Conn: session.conn, Stream: streamConn}, nil
}

func (session *MuxBindSession) Close() error {
	return session.Session.Close()
}

func (session *MuxBindSession) IsClose() bool {
	return session.Session.IsClosed()
}
