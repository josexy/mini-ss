package ss

import (
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/socks/constant"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util"
)

var defaultSSServerOpts = ssOptions{}

type ShadowsocksServer struct {
	srvs       []server.Server
	tcpRelayer *transport.TCPRelayer
	udpRelayer *transport.UDPRelayer
	Opts       ssOptions
}

func NewShadowsocksServer(opts ...SSOption) *ShadowsocksServer {
	s := &ShadowsocksServer{
		Opts: defaultSSServerOpts,
	}
	for _, o := range opts {
		o.applyTo(&s.Opts)
	}

	if len(s.Opts.serverOpts) == 0 {
		logx.Fatal("ss-server need configuration")
	}
	opt := s.Opts.serverOpts[0]
	resolver.DefaultResolver = resolver.NewDnsResolver(nil)

	switch opt.transport {
	case transport.Default:
		// default (TCP/UDP)
		// udp relay only supports the default transport
		s.srvs = append(s.srvs, server.NewTcpServer(opt.addr, s, server.Tcp).Build())
		s.srvs = append(s.srvs, server.NewUdpServer(opt.addr, s, server.Udp).Build())
	case transport.KCP:
		s.srvs = append(s.srvs, server.NewKcpServer(opt.addr, s, opt.opts).Build())
	case transport.Websocket:
		s.srvs = append(s.srvs, server.NewWsServer(opt.addr, s, opt.opts).Build())
	case transport.QUIC:
		s.srvs = append(s.srvs, server.NewQuicServer(opt.addr, s, opt.opts).Build())
	case transport.Obfs:
		s.srvs = append(s.srvs, server.NewObfsServer(opt.addr, s, opt.opts).Build())
	default:
	}

	sc, ac, err := cipher.GetCipher(opt.method, opt.password)
	if err != nil {
		logx.FatalBy(err)
	}

	tcpBound := makeStreamConn(sc, ac)
	udpBound := makePacketConn(sc, ac)

	if transport.DefaultDialerOutboundOption.AutoDetectInterface {
		if ifaceName, err := util.ResolveDefaultRouteInterface(); err == nil {
			transport.DefaultDialerOutboundOption.Interface = ifaceName
		}
	}

	s.tcpRelayer = transport.NewTCPRelayer(constant.TCPSSServerToTCPServer, transport.Default, transport.DefaultOptions, tcpBound, nil)
	s.udpRelayer = transport.NewUDPRelayer(constant.UDPSSServerToUDPServer, transport.Default, udpBound, nil)

	return s
}

func (ss *ShadowsocksServer) Start() error {

	ss.srvs = removeNilServer(ss.srvs)

	n := len(ss.srvs)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		logx.Info("start [%s] server: %s", ss.srvs[i].Type().String(), ss.srvs[i].LocalAddr())
		go ss.srvs[i].Start()
	}
	for i := 0; i < n; i++ {
		if err := <-ss.srvs[i].Error(); err != nil {
			return err
		}
	}
	return nil
}

func (ss *ShadowsocksServer) Close() error {
	n := len(ss.srvs)
	if n == 0 {
		return nil
	}
	for i := 0; i < n-1; i++ {
		ss.srvs[i].Close()
	}
	return ss.srvs[n-1].Close()
}

func (ss *ShadowsocksServer) ServeQUIC(conn net.Conn) {
	if err := ss.tcpRelayer.RelayTCP(conn, ss.Opts.serverOpts[0].addr, ""); err != nil {
		logx.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeOBFS(conn net.Conn) {
	if err := ss.tcpRelayer.RelayTCP(conn, ss.Opts.serverOpts[0].addr, ""); err != nil {
		logx.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeWS(conn net.Conn) {
	if err := ss.tcpRelayer.RelayTCP(conn, ss.Opts.serverOpts[0].addr, ""); err != nil {
		logx.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeKCP(conn net.Conn) {
	if err := ss.tcpRelayer.RelayTCP(conn, ss.Opts.serverOpts[0].addr, ""); err != nil {
		logx.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeTCP(conn net.Conn) {
	if err := ss.tcpRelayer.RelayTCP(conn, ss.Opts.serverOpts[0].addr, ""); err != nil {
		logx.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeUDP(conn net.PacketConn) {
	if err := ss.udpRelayer.RelayUDP(conn, "", ""); err != nil {
		logx.ErrorBy(err)
	}
}
