package ctxv

import (
	"github.com/josexy/mini-ss/transport"
)

type V struct {
	Addr string
	transport.Type
	transport.Options
	transport.TcpConnBound
	transport.UdpConnBound
}
