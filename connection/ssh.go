package connection

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

var _ net.Conn = (*SshConn)(nil)

type SshConn struct {
	ssh.Channel
	laddr net.Addr
	raddr net.Addr
}

func NewSshConn(channel ssh.Channel, laddr, raddr net.Addr) *SshConn {
	return &SshConn{
		Channel: channel,
		laddr:   laddr,
		raddr:   raddr,
	}
}

func (c *SshConn) LocalAddr() net.Addr { return c.laddr }

func (c *SshConn) RemoteAddr() net.Addr { return c.raddr }

func (c *SshConn) SetReadDeadline(time.Time) error { return nil }

func (c *SshConn) SetWriteDeadline(time.Time) error { return nil }

func (c *SshConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}
