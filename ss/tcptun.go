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
	"github.com/josexy/mini-ss/util"
	"github.com/josexy/mini-ss/util/ordmap"
)

type tcpTunServer struct {
	server.Server
	addr       string
	RemoteAddr string
	relayers   ordmap.OrderedMap
	ruler      *dns.Ruler
}

func newTcpTunServer(ctx context.Context, localAddr, remoteAddr string) server.Server {
	pv := ctx.Value(ctxv.SSLocalContextKey).(*ctxv.ContextPassValue)
	tt := &tcpTunServer{
		addr:       localAddr,
		RemoteAddr: remoteAddr,
		ruler:      pv.R,
	}
	pv.MAP.Range(func(name, value any) bool {
		v := value.(ctxv.V)
		tt.relayers.Store(name, transport.DstAddrRelayer{
			DstAddr:          v.Addr,
			TCPRelayer:       transport.NewTCPRelayer(constant.TCPSSLocalToSSServer, v.Type, v.Options, nil, v.TcpConnBound),
			TCPDirectRelayer: transport.NewTCPDirectRelayer(),
		})
		return true
	})

	if pv.MAP.Size() == 0 && pv.R.RuleMode == dns.Direct {
		tt.relayers.Store("", transport.DstAddrRelayer{TCPDirectRelayer: transport.NewTCPDirectRelayer()})
	}
	return tt
}

func (tt *tcpTunServer) Build() server.Server {
	tt.Server = server.NewTcpServer(tt.addr, tt, server.SimpleTcpTun)
	return tt
}

func (tt *tcpTunServer) ServeTCP(conn net.Conn) {
	host, _ := util.SplitHostPort(tt.RemoteAddr)
	if tt.ruler.Match(&host) {
		return
	}
	if err := tt.ruler.Select(&tt.relayers, conn, tt.RemoteAddr); err != nil {
		logx.ErrorBy(err)
	}
}
