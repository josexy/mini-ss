package ss

import (
	"net"

	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
)

var defaultSSServerOpts = ssOptions{}

type ShadowsocksServer struct {
	srvs       []server.Server
	tcpRelayer *relay.TCPRelayer
	udpRelayer *udpRelayer
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
		logger.Logger.Fatal("ss-server need configuration")
	}
	opt := s.Opts.serverOpts[0]
	resolver.DefaultResolver = resolver.NewDnsResolver(nil)

	sc, ac, err := cipher.GetCipher(opt.method, opt.password)
	if err != nil {
		logger.Logger.FatalBy(err)
	}

	switch opt.transport {
	case transport.Tcp:
		// udp relay only supports the default transport
		s.srvs = append(s.srvs, server.NewTcpServer(opt.addr, s, server.Tcp).Build())
		s.udpRelayer = &udpRelayer{
			addr:    opt.addr,
			relayer: relay.NewNatmapUDPRelayer(makePacketConn(sc, ac)),
		}
	case transport.Kcp:
		s.srvs = append(s.srvs, server.NewKcpServer(opt.addr, s, opt.opts).Build())
	case transport.Websocket:
		s.srvs = append(s.srvs, server.NewWsServer(opt.addr, s, opt.opts).Build())
	case transport.Quic:
		s.srvs = append(s.srvs, server.NewQuicServer(opt.addr, s, opt.opts).Build())
	case transport.Obfs:
		s.srvs = append(s.srvs, server.NewObfsServer(opt.addr, s, opt.opts).Build())
	case transport.Grpc:
		s.srvs = append(s.srvs, server.NewGrpcServer(opt.addr, s, opt.opts).Build())
	default:
	}

	s.tcpRelayer = relay.NewTCPRelayer(transport.Tcp, transport.DefaultOptions, makeStreamConn(sc, ac), nil)

	return s
}

func (ss *ShadowsocksServer) Start() error {
	n := len(ss.srvs)
	if n == 0 {
		return nil
	}

	// whether enable start udp relayer
	if ss.udpRelayer != nil {
		go func() { ss.udpRelayer.start() }()
	}

	for i := 0; i < n; i++ {
		logger.Logger.Infof("start [%s] server: %s", ss.srvs[i].Type().String(), ss.srvs[i].LocalAddr())
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
	if err := ss.tcpRelayer.RelayServerToRemote(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeOBFS(conn net.Conn) {
	if err := ss.tcpRelayer.RelayServerToRemote(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeWS(conn net.Conn) {
	if err := ss.tcpRelayer.RelayServerToRemote(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeKCP(conn net.Conn) {
	if err := ss.tcpRelayer.RelayServerToRemote(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeTCP(conn net.Conn) {
	if err := ss.tcpRelayer.RelayServerToRemote(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (ss *ShadowsocksServer) ServeGRPC(conn net.Conn) {
	if err := ss.tcpRelayer.RelayServerToRemote(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

type udpRelayer struct {
	addr    string
	relayer *relay.NatmapUDPRelayer
}

func (r *udpRelayer) start() error {
	addr, err := net.ResolveUDPAddr("udp", r.addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = r.relayer.RelayToServerToRemote(conn)
	logger.Logger.ErrorBy(err)
	return err
}
