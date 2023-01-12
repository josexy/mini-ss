package ctxv

import (
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/ordmap"
)

var SSLocalContextKey = sslocalContextKey{"ss-local-ctx-key"}

type sslocalContextKey struct{ name string }

func (c sslocalContextKey) String() string { return c.name }

type V struct {
	Addr string
	transport.Type
	transport.Options
	transport.TcpConnBound
	transport.UdpConnBound
}

type ContextPassValue struct {
	R   *dns.Ruler
	MAP ordmap.OrderedMap
}

func (cpv *ContextPassValue) Set(name string, v V) {
	cpv.MAP.Store(name, v)
}

func (cpv *ContextPassValue) Get(name string) V {
	if v, ok := cpv.MAP.Load(name); ok {
		return v.(V)
	}
	return V{}
}
