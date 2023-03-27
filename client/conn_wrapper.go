package client

import (
	"net"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/constant"
)

type tcpConnWrapper struct {
	net.Conn
	remoteAddr net.Addr // target address
}

func newTcpConnWrapper(conn net.Conn, target string) (*tcpConnWrapper, error) {
	addr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		return nil, err
	}
	return &tcpConnWrapper{
		Conn:       conn,
		remoteAddr: addr,
	}, nil
}

func (c *tcpConnWrapper) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *tcpConnWrapper) TCP() *net.TCPConn {
	return c.Conn.(*net.TCPConn)
}

func (c *tcpConnWrapper) UDP() *net.UDPConn {
	return nil
}

type udpConnWrapper struct {
	net.PacketConn
	destAddr *net.UDPAddr
	// remote target udp server address
	remoteAddr net.Addr
	buf        []byte
}

func newUdpConnWrapper(conn net.PacketConn, destAddr, target string) (*udpConnWrapper, error) {
	addr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return nil, err
	}
	laddr, err := net.ResolveUDPAddr("udp", destAddr)
	if err != nil {
		return nil, err
	}
	return &udpConnWrapper{
		PacketConn: conn,
		destAddr:   laddr,
		remoteAddr: addr,
		buf:        make([]byte, constant.MaxUdpBufferSize),
	}, nil
}

func (c *udpConnWrapper) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *udpConnWrapper) TCP() *net.TCPConn {
	return nil
}

func (c *udpConnWrapper) UDP() *net.UDPConn {
	return c.PacketConn.(*net.UDPConn)
}

func (c *udpConnWrapper) Read(b []byte) (int, error) {
	buf := c.buf

	n, _, err := c.ReadFrom(buf)
	if err != nil {
		return n, err
	}
	addr := address.ParseAddress3(buf[3:n])
	return copy(b, buf[3+len(addr):n]), nil
}

func (c *udpConnWrapper) Write(b []byte) (int, error) {
	buf := c.buf

	addr := address.ParseAddress1(c.remoteAddr.String())
	buf[0], buf[1], buf[2] = 0, 0, 0
	copy(buf[3:], addr)
	copy(buf[3+len(addr):], b)
	return c.WriteTo(buf[:3+len(addr)+len(b)], c.destAddr)
}
