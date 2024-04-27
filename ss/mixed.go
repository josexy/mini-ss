package ss

import (
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/server"
)

type mixedServer struct {
	server.Server
	addr     string
	socksSrv *socks5Server
	httpSrv  *httpProxyServer
	err      chan error
}

// newMixedServer mixed proxy mode does not support SOCKS and HTTP authentication
func newMixedServer(addr string) server.Server {
	ms := &mixedServer{
		addr:     addr,
		socksSrv: newSocksProxyServer(addr, nil),
		httpSrv:  newHttpProxyServer(addr, nil),
		err:      make(chan error, 1),
	}
	ms.Server = server.NewTcpServer(addr, server.TcpHandlerFunc(ms.handleTCPConn), server.Mixed)
	return ms
}

func (s *mixedServer) handleTCPConn(conn net.Conn) {
	bufConn := connection.NewBufioConn(conn)
	// check SOCKS or HTTP request
	data, err := bufConn.Peek(1)
	if err != nil {
		return
	}
	if len(data) != 1 {
		return
	}
	switch data[0] {
	case 0x05:
		s.socksSrv.ServeTCP(bufConn)
	default:
		s.httpSrv.ServeTCP(bufConn)
	}
}
