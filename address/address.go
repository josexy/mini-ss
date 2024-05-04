package address

import (
	"errors"
	"io"
	"net"
	"strconv"
)

var (
	errAddrTypeNotSupported = errors.New("address type not supported")
	errDomainLengthToLong   = errors.New("domain length to long")
	errInvalidHostOrPort    = errors.New("invalid host or port")
)

type Address []byte

func ParseAddressFromReader(r io.Reader, b []byte) (Address, error) {
	if len(b) < 1 {
		return nil, io.ErrShortBuffer
	}
	_, err := io.ReadFull(r, b[:1])
	if err != nil {
		return nil, err
	}

	var addrLen int
	switch b[0] {
	case 0x3: // domain name
		_, err = io.ReadFull(r, b[1:2]) // domain name length
		if err != nil {
			return nil, err
		}
		_, err = io.ReadFull(r, b[2:2+int(b[1])+2]) // domain name + port
		addrLen = 1 + 1 + int(b[1]) + 2
	case 0x1: // ipv4
		_, err = io.ReadFull(r, b[1:1+net.IPv4len+2]) // ipv4 + port
		addrLen = 1 + net.IPv4len + 2
	case 0x4: // ipv6
		_, err = io.ReadFull(r, b[1:1+net.IPv6len+2]) // ipv6 + port
		addrLen = 1 + net.IPv6len + 2
	default:
		err = errAddrTypeNotSupported
	}
	if err != nil {
		return nil, err
	}
	return b[:addrLen], err
}

func ParseAddressFromBuffer(b []byte) (Address, error) {
	addrLen := 1
	if len(b) < addrLen {
		return nil, io.ErrShortBuffer
	}

	switch b[0] {
	case 0x3:
		if len(b) < 2 {
			return nil, io.ErrShortBuffer
		}
		addrLen = 1 + 1 + int(b[1]) + 2
	case 0x1:
		addrLen = 1 + net.IPv4len + 2
	case 0x4:
		addrLen = 1 + net.IPv6len + 2
	default:
		return nil, errAddrTypeNotSupported

	}
	if len(b) < addrLen {
		return nil, io.ErrShortBuffer
	}

	return b[:addrLen], nil
}

func ParseAddressFromHostPort(host string, port int, b []byte) (Address, error) {
	if len(host) == 0 || port < 0 || port > 65535 {
		return nil, errInvalidHostOrPort
	}
	var portindex int
	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			if len(b) < 1+net.IPv4len+2 {
				return nil, io.ErrShortBuffer
			}
			b[0] = 0x1
			copy(b[1:], ip4)
			portindex = 1 + net.IPv4len
		} else {
			if len(b) < 1+net.IPv6len+2 {
				return nil, io.ErrShortBuffer
			}
			b[0] = 0x4
			copy(b[1:], ip.To16())
			portindex = 1 + net.IPv6len
		}
	} else {
		if len(host) > 255 {
			return nil, errDomainLengthToLong
		}
		if len(b) < 1+1+len(host)+2 {
			return nil, io.ErrShortBuffer
		}
		b[0], b[1] = 0x3, byte(len(host))
		copy(b[2:], host)
		portindex = 1 + 1 + len(host)
	}

	b[portindex], b[portindex+1] = byte(port>>8), byte(port)
	return b[:portindex+2], nil
}

func ParseAddress(addr string, b []byte) (Address, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	portnum, _ := strconv.Atoi(port)
	return ParseAddressFromHostPort(host, portnum, b)
}

func (a Address) String() string {
	return net.JoinHostPort(a.Host(), strconv.Itoa(a.Port()))
}

func (a Address) Host() string {
	var host string
	n := len(a)
	if n < 1 {
		return host
	}
	switch a[0] {
	case 0x3:
		if n > 1 && n >= 2+int(a[1]) {
			host = string(a[2 : 2+int(a[1])])
		}
	case 0x1:
		if n >= 1+net.IPv4len {
			host = net.IP(a[1 : 1+net.IPv4len]).String()
		}
	case 0x4:
		if n >= 1+net.IPv6len {
			host = net.IP(a[1 : 1+net.IPv6len]).String()
		}
	}
	return host
}

func (a Address) Port() int {
	var port int
	n := len(a)
	if n < 1 {
		return port
	}
	switch a[0] {
	case 0x3:
		if n > 1 && n >= 4+int(a[1]) {
			port = (int(a[2+int(a[1])]) << 8) | int(a[2+int(a[1])+1])
		}
	case 0x1:
		if n >= 3+net.IPv4len {
			port = (int(a[1+net.IPv4len]) << 8) | int(a[1+net.IPv4len+1])
		}
	case 0x4:
		if n >= 3+net.IPv6len {
			port = (int(a[1+net.IPv6len]) << 8) | int(a[1+net.IPv6len+1])
		}
	}
	return port
}
