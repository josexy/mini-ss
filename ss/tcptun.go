package ss

import (
	"net"

	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/util/logger"
)

type tcpTunServer struct {
	server.Server
	addr       string
	RemoteAddr string
}

func newTcpTunServer(localAddr, remoteAddr string) server.Server {
	return &tcpTunServer{
		addr:       localAddr,
		RemoteAddr: remoteAddr,
	}
}

func (tt *tcpTunServer) Build() server.Server {
	tt.Server = server.NewTcpServer(tt.addr, tt, server.SimpleTcpTun)
	return tt
}

func (tt *tcpTunServer) ServeTCP(conn net.Conn) {
	host, _, _ := net.SplitHostPort(tt.RemoteAddr)
	if !rule.MatchRuler.Match(&host) {
		return
	}
	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}
	if err = selector.ProxySelector.Select(proxy)(conn, tt.RemoteAddr); err != nil {
		logger.Logger.ErrorBy(err)
	}
}
