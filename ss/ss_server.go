package ss

import (
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/relay"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/netstackgo/iface"
)

var defaultSSServerOpts = ssOptions{}

type ShadowsocksServer struct {
	handlerList []*serverHandler
	srvGroup    *server.ServerGroup
	Opts        ssOptions
}

func NewShadowsocksServer(opts ...SSOption) *ShadowsocksServer {
	s := &ShadowsocksServer{
		srvGroup: server.NewServerGroup(),
		Opts:     defaultSSServerOpts,
	}
	for _, o := range opts {
		o.applyTo(&s.Opts)
	}
	if len(s.Opts.serverOpts) == 0 {
		logger.Logger.Fatal("ss-server need configuration")
	}
	resolver.DefaultResolver = resolver.NewDnsResolver(nil)
	for _, opt := range s.Opts.serverOpts {
		if err := s.initServerHandler(&opt); err != nil {
			logger.Logger.Error("init server failed", logx.Error("error", err))
		}
	}
	// check whether support auto-detect-interface
	if options.DefaultOptions.AutoDetectInterface {
		if ifaceName, err := iface.DefaultRouteInterface(); err == nil {
			options.DefaultOptions.OutboundInterface = ifaceName
			logger.Logger.Infof("auto detect outbound interface: %s", ifaceName)
		}
	}
	return s
}

func (ss *ShadowsocksServer) initServerHandler(opt *serverOptions) error {
	sc, ac, err := cipher.GetCipher(opt.method, opt.password)
	if err != nil {
		return err
	}

	handler := &serverHandler{}
	switch opt.transport {
	case transport.Tcp:
		ss.srvGroup.AddServer(server.NewTcpServer(opt.addr, handler, server.Tcp))
	case transport.Websocket:
		ss.srvGroup.AddServer(server.NewWsServer(opt.addr, handler, opt.opts))
	case transport.Quic:
		ss.srvGroup.AddServer(server.NewQuicServer(opt.addr, handler, opt.opts))
	case transport.Obfs:
		ss.srvGroup.AddServer(server.NewObfsServer(opt.addr, handler, opt.opts))
	case transport.Grpc:
		ss.srvGroup.AddServer(server.NewGrpcServer(opt.addr, handler, opt.opts))
	case transport.Ssh:
		ss.srvGroup.AddServer(server.NewSshServer(opt.addr, handler, opt.opts))
	default:
	}

	handler.tcpRelayer = relay.NewProxyTCPRelayer("", transport.Tcp, options.DefaultOptions, makeStreamConn(sc, ac), nil)
	if opt.udp {
		handler.udpRelayer = &udpRelayer{
			addr:    opt.addr,
			relayer: relay.NewNatmapUDPRelayer(makePacketConn(sc, ac), nil),
		}
	}
	ss.handlerList = append(ss.handlerList, handler)
	return nil
}

func (ss *ShadowsocksServer) Start() error {
	if ss.srvGroup.Len() == 0 {
		return nil
	}

	for _, handler := range ss.handlerList {
		// whether enable start udp relayer
		if handler.udpRelayer != nil {
			go func() { handler.udpRelayer.start() }()
		}
	}

	if err := ss.srvGroup.Start(); err != nil {
		return err
	}

	return nil
}

func (ss *ShadowsocksServer) Close() error {
	if ss.srvGroup.Len() == 0 {
		return nil
	}
	if err := ss.srvGroup.Close(); err != nil {
		return err
	}
	return nil
}

type serverHandler struct {
	tcpRelayer *relay.ProxyTCPRelayer
	udpRelayer *udpRelayer
}

func (h *serverHandler) ServeQUIC(conn net.Conn) {
	if err := h.tcpRelayer.RelayToServer(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (h *serverHandler) ServeOBFS(conn net.Conn) {
	if err := h.tcpRelayer.RelayToServer(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (h *serverHandler) ServeWS(conn net.Conn) {
	if err := h.tcpRelayer.RelayToServer(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (h *serverHandler) ServeTCP(conn net.Conn) {
	if err := h.tcpRelayer.RelayToServer(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (h *serverHandler) ServeGRPC(conn net.Conn) {
	if err := h.tcpRelayer.RelayToServer(conn); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (h *serverHandler) ServeSSH(conn net.Conn) {
	if err := h.tcpRelayer.RelayToServer(conn); err != nil {
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
	logger.Logger.Info("udp relayer", logx.String("listen", r.addr))
	err = r.relayer.RelayToServer(conn)
	logger.Logger.ErrorBy(err)
	return err
}
