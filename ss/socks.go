package ss

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"strconv"

	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/proxy"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
)

const (
	CONNECT = iota + 1
	BIND
	UDP
)

var (
	errVersion5Invalid         = errors.New("socks version not 0x05")
	errVersion1Invalid         = errors.New("socks version not 0x01")
	errUnsupportedMethod       = errors.New("socks unsupported method")
	errUnsupportedReqCmd       = errors.New("socks unsupported request cmd")
	errAuthFailure             = errors.New("socks authentication failure")
	errAuthUserShortLength     = errors.New("socks authentication user short length")
	errAuthPasswordShortLength = errors.New("socks authentication password short length")
)

type socks5Server struct {
	server.Server
	addr        string
	socksAuth   *Auth
	mitmHandler proxy.MitmHandler
	pool        *bufferpool.BufferPool
}

func newSocksProxyServer(addr string, socksAuth *Auth) *socks5Server {
	ss := &socks5Server{
		addr:      addr,
		socksAuth: socksAuth,
		pool:      bufferpool.NewBufferPool(bufferpool.MaxSocksBufferSize),
	}
	ss.Server = server.NewTcpServer(addr, ss, server.Socks)
	return ss
}

func (s *socks5Server) WithMitmMode(opt proxy.MimtOption) *socks5Server {
	var err error
	s.mitmHandler, err = proxy.NewMitmHandler(opt)
	if err != nil {
		logger.Logger.ErrorBy(err)
	}
	return s
}

func (s *socks5Server) ServeTCP(conn net.Conn) {
	dstAddr, cmd, err := s.handshake(conn)
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}
	if cmd == CONNECT {
		// TODO: in mitm mode, the client doesn't relay the data to remote ss server via transport
		if s.mitmHandler != nil {
			host, port, _ := net.SplitHostPort(dstAddr)
			ctx := context.WithValue(context.Background(), proxy.ReqCtxKey, proxy.ReqContext{
				ConnMethod: true, // ConnMethod is true for socks5 proxy
				Host:       host,
				Port:       port,
				Addr:       dstAddr,
			})
			if err = s.mitmHandler.HandleMIMT(ctx, conn); err != nil {
				logger.Logger.ErrorBy(err)
			}
			return
		}

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

		if err = selector.ProxySelector.Select(proxy).Invoke(conn, dstAddr); err != nil {
			logger.Logger.ErrorBy(err)
		}
	}
}

func (s *socks5Server) negotiate(conn net.Conn) (err error) {
	buf := s.pool.Get()
	defer s.pool.Put(buf)

	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+

	if _, err := io.ReadFull(conn, (*buf)[:1]); err != nil || (*buf)[0] != 0x05 {
		return errVersion5Invalid
	}
	if _, err = io.ReadFull(conn, (*buf)[:1]); err != nil || (*buf)[0] <= 0 {
		return errUnsupportedMethod
	}
	if _, err = io.ReadFull(conn, (*buf)[:(*buf)[0]]); err != nil {
		return err
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

func (s *socks5Server) auth(conn net.Conn, buf *[]byte) (err error) {
	// +----+------+----------+------+----------+
	// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
	// +----+------+----------+------+----------+
	// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
	// +----+------+----------+------+----------+

	var username, password string
	var userLen, passLen int
	if _, err = io.ReadFull(conn, (*buf)[:1]); err != nil || (*buf)[0] != 0x1 {
		return errVersion1Invalid
	}
	if _, err = io.ReadFull(conn, (*buf)[:1]); err != nil {
		return err
	}
	if userLen = int((*buf)[0]); userLen <= 0 {
		return errAuthUserShortLength
	}
	if _, err = io.ReadFull(conn, (*buf)[:userLen]); err != nil {
		return err
	}
	username = string((*buf)[:userLen])
	if _, err = io.ReadFull(conn, (*buf)[:1]); err != nil {
		return err
	}
	if passLen = int((*buf)[0]); passLen <= 0 {
		return errAuthPasswordShortLength
	}
	if _, err = io.ReadFull(conn, (*buf)[:passLen]); err != nil {
		return err
	}
	password = string((*buf)[:passLen])

	// +----+--------+
	// |VER | STATUS |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	(*buf)[0] = 0x01
	if s.socksAuth.Validate(username, password) {
		(*buf)[1] = 0x00
		conn.Write((*buf)[:2])
		return nil
	}
	(*buf)[1] = 0x01
	conn.Write((*buf)[:2])
	return errAuthFailure
}

func (s *socks5Server) request(conn net.Conn) (addr string, cmd byte, err error) {
	buf := s.pool.Get()

	// var n int
	// n, err = conn.Read(*buf)
	// if err != nil {
	// 	s.pool.Put(buf)
	// 	return
	// }
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	if _, err = io.ReadFull(conn, (*buf)[:1]); err != nil || (*buf)[0] != 0x05 {
		s.pool.Put(buf)
		err = errVersion5Invalid
		return
	}
	if _, err = io.ReadFull(conn, (*buf)[:2]); err != nil {
		s.pool.Put(buf)
		return
	}
	cmd = (*buf)[0]
	dstAddr, err := address.ParseAddressFromReader(conn, *buf)
	if err != nil {
		s.pool.Put(buf)
		return
	}
	// the host may be a domain name, fake ip address or real ip address
	host := dstAddr.Host()
	// if tun mode is enabled, the host may be a fake ip address
	// so we need to resolve the domain name
	if resolver.DefaultResolver.IsEnhancerMode() {
		if ip, e := netip.ParseAddr(host); e == nil && resolver.DefaultResolver.IsFakeIP(ip) {
			if record, e := resolver.DefaultResolver.FindByIP(ip); e == nil {
				// fetch the domain name from the fake ip
				host = record.Domain
			} else {
				err = e
				return
			}
		}
	}
	// the host may be a domain name or a real ip address
	if !rule.MatchRuler.Match(&host) {
		s.pool.Put(buf)
		s.handleFail(conn, 0x02)
		err = rule.ErrRuleMatchDropped
		return
	}
	addr = net.JoinHostPort(host, strconv.FormatInt(int64(dstAddr.Port()), 10))

	switch cmd {
	case CONNECT:
		if err = s.handleCmdConnect(conn, buf); err != nil {
			return
		}
	case UDP:
		if err = s.handleCmdUdpAssociate(conn, buf); err != nil {
			return
		}
	default:
		s.pool.Put(buf)
		s.handleFail(conn, 0x07)
		err = errUnsupportedReqCmd
		return
	}
	return
}

func (s *socks5Server) handshake(conn net.Conn) (dstAddr string, cmd byte, err error) {
	if err = s.negotiate(conn); err != nil {
		logger.Logger.ErrorBy(err)
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
	bindAddr, err := address.ParseAddressFromHostPort("0.0.0.0", 0, make([]byte, 7))
	if err != nil {
		return err
	}
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

	dstConn, err := transport.ListenLocalUDP(context.Background())
	if err != nil {
		return err
	}
	defer dstConn.Close()

	(*buf)[0], (*buf)[1], (*buf)[2] = 0x05, 0x00, 0x00

	_, port, _ := net.SplitHostPort(dstConn.LocalAddr().String())
	bindAddr, _ := address.ParseAddress(net.JoinHostPort("127.0.0.1", port), make([]byte, 7))
	logger.Logger.Tracef("socks5 udp associate bind udp address: %s", bindAddr.String())

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
	return selector.ProxySelector.SelectPacket(proxy).Invoke(dstConn, "")
}

func (s *socks5Server) handleFail(conn net.Conn, errno byte) {
	conn.Write([]byte{0x05, errno, 0x00})
}
