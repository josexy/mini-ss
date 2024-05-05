package ctxv

import (
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/transport"
)

type V struct {
	Addr string
	transport.Type
	transport.TcpConnBound
	transport.UdpConnBound
	options.Options
}
