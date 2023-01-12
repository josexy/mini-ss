package address

import (
	"errors"
	"io"
	"net"
	"strconv"

	"github.com/josexy/mini-ss/bufferpool"
)

const maxAddrLen = 1 + 1 + 255 + 2

var pool = bufferpool.NewBufferPool(maxAddrLen)

type Address []byte

func getAddrBuf() *[]byte { return pool.Get() }

func PutAddrBuf(buf *[]byte) { pool.Put(buf) }

func ParseAddress4(r io.Reader) (Address, *[]byte, error) {
	rbuf := getAddrBuf()
	b := *rbuf
	// target address type
	_, err := io.ReadFull(r, b[:1])
	if err != nil {
		PutAddrBuf(rbuf)
		return nil, nil, err
	}
	addrLen := 1

	switch b[0] {
	case 0x3: // domain name
		_, err = io.ReadFull(r, b[1:2]) // domain name length
		if err != nil {
			PutAddrBuf(rbuf)
			return nil, nil, err
		}
		_, err = io.ReadFull(r, b[2:2+int(b[1])+2])
		addrLen = 1 + 1 + int(b[1]) + 2
	case 0x1: // ipv4
		_, err = io.ReadFull(r, b[1:1+net.IPv4len+2])
		addrLen = 1 + net.IPv4len + 2
	case 0x4: // ipv6
		_, err = io.ReadFull(r, b[1:1+net.IPv6len+2])
		addrLen = 1 + net.IPv6len + 2
	default:
		err = errors.New("address type not support")
	}
	if err != nil {
		PutAddrBuf(rbuf)
		return nil, nil, err
	}
	return b[:addrLen], rbuf, err
}

func ParseAddress3(b []byte) Address {
	addrLen := 1
	if len(b) < addrLen {
		return nil
	}

	switch b[0] {
	case 0x3:
		if len(b) < 2 {
			return nil
		}
		addrLen = 1 + 1 + int(b[1]) + 2
	case 0x1:
		addrLen = 1 + net.IPv4len + 2
	case 0x4:
		addrLen = 1 + net.IPv6len + 2
	default:
		return nil

	}

	if len(b) < addrLen {
		return nil
	}

	return b[:addrLen]
}

func ParseAddress0(host string, port int) Address {
	var buf Address
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			buf = make([]byte, 1+net.IPv4len+2)
			buf[0] = 0x1
			copy(buf[1:], ip4)
		} else {
			buf = make([]byte, 1+net.IPv6len+2)
			buf[0] = 0x4
			copy(buf[1:], ip)
		}
	} else {
		if len(host) > 255 {
			return nil
		}
		buf = make([]byte, 1+1+len(host)+2)
		buf[0] = 0x3
		buf[1] = byte(len(host))
		copy(buf[2:], host)
	}

	buf[len(buf)-2], buf[len(buf)-1] = byte(port>>8), byte(port)
	return buf
}

func ParseAddress1(addr string) Address {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}
	portnum, _ := strconv.Atoi(port)
	return ParseAddress0(host, portnum)
}

func (a Address) String() string {
	host := a.Host()
	port := a.Port()
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func (a Address) Host() string {
	var host string
	if len(a) < 1 {
		return host
	}
	switch a[0] {
	case 0x3:
		host = string(a[2 : 2+int(a[1])])
	case 0x1:
		host = net.IP(a[1 : 1+net.IPv4len]).String()
	case 0x4:
		host = net.IP(a[1 : 1+net.IPv6len]).String()
	}
	return host
}

func (a Address) Port() int {
	var port int
	if len(a) < 1 {
		return port
	}
	switch a[0] {
	case 0x3:
		port = (int(a[2+int(a[1])]) << 8) | int(a[2+int(a[1])+1])
	case 0x1:
		port = (int(a[1+net.IPv4len]) << 8) | int(a[1+net.IPv4len+1])
	case 0x4:
		port = (int(a[1+net.IPv6len]) << 8) | int(a[1+net.IPv6len+1])
	}
	return port
}
