package ss

import (
	"bufio"
	"net"

	"github.com/josexy/mini-ss/server"
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
func newMixedServer(addr string) server.Server {
	return &mixedServer{
		addr:     addr,
		socksSrv: newSocksProxyServer(addr, nil),
		httpSrv:  newHttpProxyServer(addr, nil),
		err:      make(chan error, 1),
	}
}

func (s *mixedServer) Build() server.Server {
	// rewrite tcp connection inbound
	s.Server = server.NewTcpServer(s.addr, server.TcpHandlerFunc(s.handleTCPConn), server.Mixed)
	return s
}

func (s *mixedServer) Start() {
	s.Server.Start()
}

func (s *mixedServer) Error() chan error {
	return s.Server.Error()
}

func (s *mixedServer) Close() error {
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
	case 0x05:
		s.socksSrv.ServeTCP(br)
	default:
		s.httpSrv.ServeTCP(br)
	}
}
