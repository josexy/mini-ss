package ss

import (
	"io"
	"net"
	"net/netip"
	"strconv"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
)

type socks5Server struct {
	server.Server
	addr      string
	socksAuth *Auth
	pool      *bufferpool.BufferPool
}

func newSocksProxyServer(addr string, socksAuth *Auth) *socks5Server {
	ss := &socks5Server{
		addr:      addr,
		socksAuth: socksAuth,
		pool:      bufferpool.NewBufferPool(constant.MaxSocksBufferSize),
	}
	ss.Server = server.NewTcpServer(addr, ss, server.Socks)
	return ss
}

func (s *socks5Server) ServeTCP(conn net.Conn) {
	dstAddr, cmd, err := s.handshake(conn)
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}
	if cmd == constant.Connect {
		proxy, err := rule.MatchRuler.Select()
		if err != nil {
			logger.Logger.ErrorBy(err)
			return
		}

		if statistic.EnableStatistic {
			tcpTracker := statistic.NewTCPTracker(conn, statistic.Context{
				Src:     conn.RemoteAddr().String(),
				Dst:     dstAddr,
				Network: "TCP",
				Type:    "SOCKS",
				Proxy:   proxy,
				Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
			})
			defer statistic.DefaultManager.Remove(tcpTracker)
			conn = tcpTracker
		}

		if err = selector.ProxySelector.Select(proxy)(conn, dstAddr); err != nil {
			logger.Logger.ErrorBy(err)
		}
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
	if version != 0x05 {
		return constant.ErrVersion5Invalid
	}
	if nMethods < 0 {
		return constant.ErrUnsupportedMethod
	}

	method := 0x00
	if s.socksAuth != nil {
		method = 0x02
	}
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	(*buf)[0], (*buf)[1] = 0x05, byte(method)
	conn.Write((*buf)[:2])

	if method == 0x02 {
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

	if version != 0x01 {
		return constant.ErrVersion1Invalid
	}

	// +----+--------+
	// |VER | STATUS |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	(*buf)[0] = 0x01
	if s.socksAuth != nil && s.socksAuth.Validate(uName, passWd) {
		(*buf)[1] = 0x00
		conn.Write((*buf)[:2])
		return nil
	}
	(*buf)[1] = 0x01
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

	if version != 0x05 {
		s.pool.Put(buf)
		err = constant.ErrVersion5Invalid
		return
	}
	cmd = _cmd

	// the host may be a fake ip address or real ip address
	host := dstAddr.Host()
	// if tun mode is enabled, the host may be a fake ip address
	// so we need to resolve the domain name
	if resolver.DefaultResolver.IsEnhancerMode() {
		if ip, err := netip.ParseAddr(host); err == nil {
			record := resolver.DefaultResolver.FindByIP(ip)
			if record != nil {
				// fetch the domain name from the fake ip
				host = record.Domain
			}
		}
	}
	// the host may be a domain name or a real ip address
	if !rule.MatchRuler.Match(&host) {
		s.handleFail(conn, 0x02)
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
	default:
		s.pool.Put(buf)
		s.handleFail(conn, 0x07)
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

	(*buf)[0], (*buf)[1], (*buf)[2] = 0x05, 0x00, 0x00
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

	dstConn, err := transport.ListenLocalUDP()
	if err != nil {
		return err
	}
	defer dstConn.Close()

	(*buf)[0], (*buf)[1], (*buf)[2] = 0x05, 0x00, 0x00

	_, port, _ := net.SplitHostPort(dstConn.LocalAddr().String())
	bindAddr := address.ParseAddress1(net.JoinHostPort("127.0.0.1", port))
	logger.Logger.Tracef("socks5 bind udp address: %s", bindAddr.String())

	copy((*buf)[3:], bindAddr)
	conn.Write((*buf)[:3+len(bindAddr)])
	s.pool.Put(buf)

	go func() {
		// wait for the tcp connection to close
		buf := make([]byte, 1)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return
			}
		}
	}()

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		return err
	}
	if statistic.EnableStatistic {
		udpTracker := statistic.NewUDPTracker(dstConn, statistic.Context{
			Src:     bindAddr.String(),
			Dst:     "-",
			Network: "UDP",
			Type:    "SOCKS",
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
			Proxy:   proxy,
		})
		defer statistic.DefaultManager.Remove(udpTracker)
		dstConn = udpTracker
	}
	return selector.ProxySelector.SelectPacket(proxy)(dstConn, "")
}

func (s *socks5Server) handleFail(conn net.Conn, errno byte) {
	conn.Write([]byte{0x05, errno, 0x00})
}
