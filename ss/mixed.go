package ss

import (
	"bufio"
	"context"
	"net"

	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/socks/constant"
)

type bufferConn struct {
	net.Conn
	r *bufio.Reader
}

func newBufferConn(c net.Conn) *bufferConn { return &bufferConn{Conn: c, r: bufio.NewReader(c)} }

func (c *bufferConn) Peek(n int) ([]byte, error) { return c.r.Peek(n) }

func (c *bufferConn) Read(p []byte) (int, error) { return c.r.Read(p) }

type mixedServer struct {
	server.Server
	addr     string
	socksSrv *socks5Server
	httpSrv  *httpProxyServer
	err      chan error
}

// newMixedServer mixed proxy mode does not support SOCKS and HTTP authentication
func newMixedServer(ctx context.Context, addr string) server.Server {
	return &mixedServer{
		addr:     addr,
		socksSrv: newSocksProxyServer(ctx, addr, nil),
		httpSrv:  newHttpProxyServer(ctx, addr, nil),
		err:      make(chan error, 1),
	}
}

func (s *mixedServer) Build() server.Server {
	if s.socksSrv.udp {
		s.socksSrv.udpSrv = server.NewUdpServer(s.addr, s.socksSrv, server.Socks)
	}
	// rewrite tcp connection inbound
	s.Server = server.NewTcpServer(s.addr, server.TcpHandlerFunc(s.handleTCPConn), server.Mixed)
	return s
}

func (s *mixedServer) Start() {
	if s.socksSrv.udp {
		go s.socksSrv.udpSrv.Start()
	}
	s.Server.Start()
}

func (s *mixedServer) Error() chan error {
	if s.socksSrv.udp {
		if udpErr := <-s.socksSrv.udpSrv.Error(); udpErr != nil {
			s.err <- udpErr
			return s.err
		}
	}
	s.err = s.Server.Error()
	return s.err
}

func (s *mixedServer) Close() error {
	if s.socksSrv.udp {
		s.socksSrv.udpSrv.Close()
	}
	return s.Server.Close()
}

func (s *mixedServer) handleTCPConn(conn net.Conn) {
	br := newBufferConn(conn)
	// check SOCKS or HTTP request
	data, err := br.Peek(1)
	if err != nil {
		return
	}
	if len(data) != 1 {
		return
	}
	switch data[0] {
	case constant.Socks5Version05:
		s.socksSrv.ServeTCP(br)
	default:
		s.httpSrv.ServeTCP(br)
	}
}
