package client

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/transport"
)

var defaultSupportMethods = []byte{
	0x00, // NoAuthRequired,
	0x02, // UsernamePassword,
}

type Socks5Client struct {
	dialer     transport.Dialer
	Addr       string
	conn       net.Conn
	udpConn    net.PacketConn
	timeout    time.Duration
	authMethod byte
	authInfo   *url.Userinfo
	buf        []byte
}

func NewSocks5Client(addr string) *Socks5Client {
	return &Socks5Client{
		Addr:       addr,
		timeout:    10 * time.Second,
		authMethod: 0x00,
		buf:        make([]byte, bufferpool.MaxSocksBufferSize),
		dialer:     transport.NewDialer(transport.Tcp, options.DefaultOptions),
	}
}

func (c *Socks5Client) SetSocksAuth(username, password string) {
	c.authInfo = url.UserPassword(username, password)
	c.authMethod = 0x02
}

func (c *Socks5Client) Close() (err error) {
	if c.conn != nil {
		err = c.conn.Close()
	}
	if c.udpConn != nil {
		err = c.udpConn.Close()
	}
	return
}

func (c *Socks5Client) Dial(ctx context.Context, addr string) (transport.Conn, error) {
	// don't care the tcp connection bind address
	_, err := c.handshake(ctx, addr, 1) // CONNECT
	if err != nil {
		return nil, err
	}
	tcw, err := newTcpConnWrapper(c.conn, addr)
	c.conn = tcw
	return tcw, err
}

func (c *Socks5Client) DialUDP(ctx context.Context, addr string) (transport.Conn, error) {
	bindAddr, err := c.handshake(ctx, addr, 3) // UDP
	if err != nil {
		return nil, err
	}
	conn, err := transport.ListenLocalUDP(ctx)
	if err != nil {
		return nil, err
	}
	ucw, err := newUdpConnWrapper(conn, bindAddr, addr)
	c.udpConn = ucw
	return ucw, err
}

func (c *Socks5Client) handshake(ctx context.Context, address string, cmd byte) (string, error) {
	conn, err := c.dialer.Dial(ctx, c.Addr)
	if err != nil {
		return "", err
	}
	c.conn = conn
	if err = c.negotiate(conn); err != nil {
		_ = conn.Close()
		return "", err
	}

	if err = c.authentication(conn); err != nil {
		_ = conn.Close()
		return "", err
	}
	var bindAddr string
	if bindAddr, err = c.request(conn, address, cmd); err != nil {
		_ = conn.Close()
		return "", err
	}
	return bindAddr, nil
}

func (c *Socks5Client) negotiate(conn net.Conn) error {
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	buf := c.buf
	buf[0] = 0x05
	buf[1] = byte(len(defaultSupportMethods))
	copy(buf[2:], defaultSupportMethods)
	conn.Write(buf[:2+len(defaultSupportMethods)])

	_, err := conn.Read(buf)
	if err != nil {
		return err
	}
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	version, method := buf[0], buf[1]
	if version != 0x05 {
		return errors.New("socks version not 0x05")
	}

	c.authMethod = method
	return nil
}

func (c *Socks5Client) authentication(conn net.Conn) error {
	if c.authMethod != 0x02 {
		return nil
	}

	buf := c.buf

	// +----+------+----------+------+----------+
	// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
	// +----+------+----------+------+----------+
	// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
	// +----+------+----------+------+----------+
	buf[0] = 0x01
	buf[1] = byte(len(c.authInfo.Username()))
	nu := copy(buf[2:], c.authInfo.Username())
	p, _ := c.authInfo.Password()
	buf[2+nu] = byte(len(p))
	np := copy(buf[3+nu:], p)
	conn.Write(buf[:3+nu+np])

	// +----+--------+
	// |VER | STATUS |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	_, err := conn.Read(buf)
	if err != nil {
		return err
	}
	version, status := buf[0], buf[1]
	if version != 0x01 {
		return errors.New("socks version not 0x01")
	}
	if status != 0x00 {
		return errors.New("socks authentication failure")
	}
	return nil
}

func (c *Socks5Client) request(conn net.Conn, target string, cmd byte) (string, error) {
	var host string
	var port int
	hp := strings.Split(target, ":")
	host = hp[0]
	// HTTP request
	if len(hp) == 1 {
		port = 80
	} else {
		port, _ = strconv.Atoi(hp[1])
	}

	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	buf := c.buf
	dstAddr, err := address.ParseAddressFromHostPort(host, port, make([]byte, 259))
	if err != nil {
		return "", err
	}
	buf[0], buf[1], buf[2] = 0x05, cmd, 0
	copy(buf[3:], dstAddr)
	conn.Write(buf[:3+len(dstAddr)])

	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	_, err = conn.Read(buf)
	if err != nil {
		return "", err
	}
	_, code := buf[0], buf[1]
	if code != 0x00 {
		return "", errors.New("socks request failure")
	}
	bindAddr, err := address.ParseAddressFromBuffer(buf[3:])
	if err != nil {
		return "", err
	}
	return bindAddr.String(), nil
}
