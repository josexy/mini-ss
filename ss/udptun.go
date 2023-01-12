package ss

import (
	"context"
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/socks/constant"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/ordmap"
)

type udpTunServer struct {
	server.Server
	addr       string
	RemoteAddr string
	relayers   ordmap.OrderedMap
	ruler      *dns.Ruler
	udp        bool
}

func newUdpTunServer(ctx context.Context, localAddr, remoteAddr string) server.Server {
	pv := ctx.Value(ctxv.SSLocalContextKey).(*ctxv.ContextPassValue)
	ut := &udpTunServer{
		addr:       localAddr,
		RemoteAddr: remoteAddr,
		ruler:      pv.R,
	}
	pv.MAP.Range(func(name, value any) bool {
		v := value.(ctxv.V)
		if v.Type == transport.Default {
			ut.udp = true
			ut.relayers.Store(name, transport.DstAddrRelayer{
				DstAddr:          v.Addr,
				UDPRelayer:       transport.NewUDPRelayer(constant.UDPTunServerToSSServer, v.Type, nil, v.UdpConnBound),
				UDPDirectRelayer: transport.NewUDPDirectRelayer(),
			})
		}
		return true
	})
	if pv.MAP.Size() == 0 && pv.R.RuleMode == dns.Direct {
		ut.relayers.Store("", transport.DstAddrRelayer{UDPDirectRelayer: transport.NewUDPDirectRelayer()})
	}
	return ut
}

func (ut *udpTunServer) Build() server.Server {
	if !ut.udp {
		return nil
	}
	ut.Server = server.NewUdpServer(ut.addr, ut, server.SimpleUdpTun)
	return ut
}

func (ut *udpTunServer) ServeUDP(conn net.PacketConn) {
	if err := ut.ruler.SelectOne(&ut.relayers, conn, ut.RemoteAddr); err != nil {
		logx.ErrorBy(err)
	}
}
