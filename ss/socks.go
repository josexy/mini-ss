package ss

import (
	"context"
	"io"
	"net"
	"net/netip"
	"strconv"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/auth"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/socks/constant"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/ordmap"
)

type socks5Server struct {
	server.Server
	udpSrv    server.Server
	addr      string
	socksAuth *auth.Auth
	relayers  ordmap.OrderedMap
	pool      *bufferpool.BufferPool
	ruler     *dns.Ruler
	udp       bool
	err       chan error
}

func newSocksProxyServer(ctx context.Context, addr string, socksAuth *auth.Auth) *socks5Server {
	pv := ctx.Value(ctxv.SSLocalContextKey).(*ctxv.ContextPassValue)
	sp := &socks5Server{
		addr:      addr,
		socksAuth: socksAuth,
		ruler:     pv.R,
		pool:      bufferpool.NewBufferPool(constant.MaxBufferSize),
		err:       make(chan error, 1),
	}

	pv.MAP.Range(func(name, value any) bool {
		v := value.(ctxv.V)
		relayer := transport.DstAddrRelayer{
			DstAddr:          v.Addr,
			TCPRelayer:       transport.NewTCPRelayer(constant.TCPSSLocalToSSServer, v.Type, v.Options, nil, v.TcpConnBound),
			TCPDirectRelayer: transport.NewTCPDirectRelayer(),
		}
		// UDP relay only supports default transport type
		if v.Type == transport.Default {
			relayer.UDPRelayer = transport.NewUDPRelayer(constant.UDPSSLocalToSSServer, v.Type, nil, v.UdpConnBound)
			relayer.UDPDirectRelayer = transport.NewUDPDirectRelayer()
			sp.udp = true
		}
		sp.relayers.Store(name, relayer)
		return true
	})

	if pv.MAP.Size() == 0 && pv.R.RuleMode == dns.Direct {
		sp.relayers.Store("", transport.DstAddrRelayer{
			TCPDirectRelayer: transport.NewTCPDirectRelayer(),
			UDPDirectRelayer: transport.NewUDPDirectRelayer(),
		})
	}
	return sp
}

func (s *socks5Server) Build() server.Server {
	s.Server = server.NewTcpServer(s.addr, s, server.Socks)
	if s.udp {
		s.udpSrv = server.NewUdpServer(s.addr, s, server.Socks)
	}
	return s
}

func (s *socks5Server) Start() {
	if s.udp {
		go s.udpSrv.Start()
	}
	s.Server.Start()
}

func (s *socks5Server) Error() chan error {
	if s.udp {
		if udpErr := <-s.udpSrv.Error(); udpErr != nil {
			s.err <- udpErr
			return s.err
		}
	}
	s.err = s.Server.Error()
	return s.err
}

func (s *socks5Server) Close() error {
	if s.udp {
		s.udpSrv.Close()
	}
	return s.Server.Close()
}

func (s *socks5Server) ServeTCP(conn net.Conn) {
	dstAddr, cmd, err := s.handshake(conn)
	if err == nil && cmd == constant.Connect {
		err = s.ruler.Select(&s.relayers, conn, dstAddr)
	}
	if err != nil {
		logx.ErrorBy(err)
	}
}

func (s *socks5Server) ServeUDP(conn net.PacketConn) {
	if err := s.ruler.SelectOne(&s.relayers, conn, ""); err != nil {
		logx.ErrorBy(err)
	}
}

func (s *socks5Server) negotiate(conn net.Conn) error {
	buf := s.pool.Get()
	defer s.pool.Put(buf)

	n, err := conn.Read(*buf)
	if err != nil {
		return err
	}
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	version, nMethods, _ := (*buf)[0], int((*buf)[1]), (*buf)[2:n]
	if version != constant.Socks5Version05 {
		return constant.ErrVersion5Invalid
	}
	if nMethods < 0 {
		return constant.ErrUnsupportedMethod
	}

	method := constant.MethodNoAuthRequired
	if s.socksAuth != nil {
		method = constant.MethodUsernamePassword
	}
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	(*buf)[0], (*buf)[1] = constant.Socks5Version05, byte(method)
	conn.Write((*buf)[:2])

	if method == constant.MethodUsernamePassword {
		return s.auth(conn, buf)
	}
	return nil
}

func (s *socks5Server) auth(conn net.Conn, buf *[]byte) error {
	n, err := conn.Read(*buf)
	if err != nil {
		return err
	}
	// +----+------+----------+------+----------+
	// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
	// +----+------+----------+------+----------+
	// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
	// +----+------+----------+------+----------+
	version := (*buf)[0]
	uLen := int((*buf)[1])
	uName := string((*buf)[2 : 2+uLen])
	pLen := int((*buf)[2+uLen])
	passWd := string((*buf)[n-pLen : n])

	if version != constant.Socks5Version01 {
		return constant.ErrVersion1Invalid
	}

	// +----+--------+
	// |VER | STATUS |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	(*buf)[0] = constant.Socks5Version01
	if s.socksAuth != nil && s.socksAuth.Validate(uName, passWd) {
		(*buf)[1] = constant.Succeed
		conn.Write((*buf)[:2])
		return nil
	}
	(*buf)[1] = constant.GeneralSocksServerFailure
	conn.Write((*buf)[:2])
	return constant.ErrAuthFailure
}

func (s *socks5Server) request(conn net.Conn) (addr string, cmd byte, err error) {
	buf := s.pool.Get()

	var n int
	n, err = conn.Read(*buf)
	if err != nil {
		s.pool.Put(buf)
		return
	}
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	version, _cmd, dstAddr := (*buf)[0], (*buf)[1], address.ParseAddress3((*buf)[3:n])

	if version != constant.Socks5Version05 {
		s.pool.Put(buf)
		err = constant.ErrVersion5Invalid
		return
	}
	cmd = _cmd

	host := dstAddr.Host()
	// if tun mode is enabled, use fake dns to reverse lookup domain name from fake ip address
	if resolver.DefaultResolver.IsFakeIPMode() {
		// whether the host is a fake ip address
		if ip, err := netip.ParseAddr(host); err == nil {
			record := resolver.DefaultResolver.FindByIP(ip)
			if record != nil {
				host = record.Domain
			}
		}
	}

	if s.ruler.Match(&host) {
		s.handleFail(conn, constant.ConnectionNotAllowedByRuleset)
		err = constant.ErrRuleMatchDropped
		return
	}
	addr = net.JoinHostPort(host, strconv.FormatInt(int64(dstAddr.Port()), 10))

	switch _cmd {
	case constant.Connect:
		if err = s.handleCmdConnect(conn, buf); err != nil {
			return
		}
	case constant.UDP:
		if err = s.handleCmdUdpAssociate(conn, buf); err != nil {
			return
		}
	//case constant.Bind:
	default:
		s.pool.Put(buf)
		s.handleFail(conn, constant.CommandNotSupported)
		err = constant.ErrUnsupportedReqCmd
		return
	}
	return
}

func (s *socks5Server) handshake(conn net.Conn) (dstAddr string, cmd byte, err error) {
	if err = s.negotiate(conn); err != nil {
		return
	}
	return s.request(conn)
}

func (s *socks5Server) handleCmdConnect(conn net.Conn, buf *[]byte) error {
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	(*buf)[0] = constant.Socks5Version05
	(*buf)[1] = constant.Succeed
	(*buf)[2] = 0x00
	// the bind address and port should be empty
	bindAddr := address.ParseAddress0("0.0.0.0", 0)
	copy((*buf)[3:], bindAddr)
	conn.Write((*buf)[:3+len(bindAddr)])
	s.pool.Put(buf)
	return nil
}

func (s *socks5Server) handleCmdUdpAssociate(conn net.Conn, buf *[]byte) error {
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	(*buf)[0] = constant.Socks5Version05
	(*buf)[1] = constant.Succeed
	(*buf)[2] = 0x00
	bindAddr := address.ParseAddress1(s.addr)
	copy((*buf)[3:], bindAddr)
	conn.Write((*buf)[:3+len(bindAddr)])
	s.pool.Put(buf)

	tcpDoneChan := make(chan error)
	go func() {
		// wait for tcp connection to closed
		buf := make([]byte, 1)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				tcpDoneChan <- err
				return
			}
		}
	}()

	return <-tcpDoneChan
}

func (s *socks5Server) handleFail(conn net.Conn, errno byte) {
	conn.Write([]byte{constant.Socks5Version05, errno, 0x00})
}
