package ss

import (
	"net"
	"net/url"

	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/enhancer"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/ssr"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/netstackgo/iface"
	"github.com/josexy/netstackgo/tun"
	"github.com/josexy/proxyutil"
)

var defaultSSLocalOpts = ssOptions{
	localOpts: localOptions{
		enhancerConfig: enhancer.EnhancerConfig{
			Tun: tun.TunConfig{
				Name: "utun3",
				Addr: "198.18.0.1/16",
				MTU:  tun.DefaultMTU,
			},
		},
	},
}

type ShadowsocksClient struct {
	srvs     []server.Server
	Opts     ssOptions
	enhancer *enhancer.Enhancer
}

func NewShadowsocksClient(opts ...SSOption) *ShadowsocksClient {
	s := &ShadowsocksClient{
		Opts: defaultSSLocalOpts,
	}
	for _, o := range opts {
		o.applyTo(&s.Opts)
	}

	// whether to support auto-detect-interface
	if transport.DefaultDialerOutboundOption.AutoDetectInterface {
		if ifaceName, err := iface.DefaultRouteInterface(); err == nil {
			transport.DefaultDialerOutboundOption.Interface = ifaceName
			logger.Logger.Infof("auto detect outbound interface: %q", ifaceName)
		}
	}

	// init the global default dns resolver
	resolver.DefaultResolver = resolver.NewDnsResolver(dns.DefaultDnsNameservers)

	// only one proxy node
	if len(s.Opts.serverOpts) == 1 {
		s.Opts.serverOpts[0].name = "<Default>"
		rule.MatchRuler.GlobalTo = "<Default>"
	}

	for _, opt := range s.Opts.serverOpts {
		sc, ac, err := cipher.GetCipher(opt.method, opt.password)
		if err != nil {
			logger.Logger.FatalBy(err)
		}
		var tcpBound transport.TcpConnBound
		var udpBound transport.UdpConnBound

		// whether to support shadowsocksr
		if !opt.ssr {
			tcpBound = makeStreamConn(sc, ac)
			udpBound = makePacketConn(sc, ac)
		} else {
			cp, err := ssr.NewSSRClientStreamCipher(sc,
				opt.addr,                                      // host,port
				opt.ssrOpt.Protocol, opt.ssrOpt.ProtocolParam, // protocol,protocol-param
				opt.ssrOpt.Obfs, opt.ssrOpt.ObfsParam) // obfs,obfs-param

			if err != nil {
				logger.Logger.FatalBy(err)
			}

			tcpBound = makeSSRClientStreamConn(cp)
			udpBound = makeSSRClientPacketConn(cp)
		}
		item := ctxv.V{
			Addr:         opt.addr,
			Options:      opt.opts,
			Type:         opt.transport,
			TcpConnBound: tcpBound,
			UdpConnBound: udpBound,
		}
		selector.ProxySelector.AddProxy(opt.name, item)
		// enable udp relay
		if opt.udp {
			selector.ProxySelector.AddPacketProxy(opt.name, item)
		}
	}
	// create simple tcp tun server
	for _, addrs := range s.Opts.localOpts.tcpTunAddr {
		s.srvs = append(s.srvs, newTcpTunServer(addrs[0], addrs[1]).Build())
	}

	// enable mixed proxy
	if s.Opts.localOpts.mixedAddr != "" {
		s.srvs = append(s.srvs, newMixedServer(s.Opts.localOpts.mixedAddr).Build())
	} else {
		if s.Opts.localOpts.httpAddr != "" {
			// http proxy
			s.srvs = append(s.srvs, newHttpProxyServer(s.Opts.localOpts.httpAddr, s.Opts.localOpts.httpAuth).Build())
		}
		if s.Opts.localOpts.socksAddr != "" {
			// socks proxy
			s.srvs = append(s.srvs, newSocksProxyServer(s.Opts.localOpts.socksAddr, s.Opts.localOpts.socksAuth).Build())
		}
	}

	if s.Opts.localOpts.enableTun {
		s.enhancer = enhancer.NewEnhancer(s.Opts.localOpts.enhancerConfig)
	}
	return s
}

func (ss *ShadowsocksClient) setSystemProxy() {

	var http, socks *url.URL

	// using mixed proxy instead of SOCKS and HTTP proxy
	if ss.Opts.localOpts.mixedAddr != "" {
		_, port, _ := net.SplitHostPort(ss.Opts.localOpts.mixedAddr)
		http = &url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort("127.0.0.1", port),
		}
		socks = &url.URL{
			Scheme: "socks",
			Host:   net.JoinHostPort("127.0.0.1", port),
		}
	} else {
		if ss.Opts.localOpts.httpAddr != "" {
			_, port1, _ := net.SplitHostPort(ss.Opts.localOpts.httpAddr)
			http = &url.URL{
				Scheme: "http",
				Host:   net.JoinHostPort("127.0.0.1", port1),
				User:   ss.Opts.localOpts.httpAuth.UserInfo(),
			}
		}
		if ss.Opts.localOpts.socksAddr != "" {
			_, port2, _ := net.SplitHostPort(ss.Opts.localOpts.socksAddr)
			socks = &url.URL{
				Scheme: "socks",
				Host:   net.JoinHostPort("127.0.0.1", port2),
				User:   ss.Opts.localOpts.socksAuth.UserInfo(),
			}
		}
	}
	proxyutil.SetSystemProxy(http, socks)
}

func (ss *ShadowsocksClient) initEnhancer() error {
	if ss.Opts.localOpts.enableTun {
		return ss.enhancer.Start()
	}
	return nil
}

func (ss *ShadowsocksClient) closeTun() error {
	if ss.Opts.localOpts.enableTun {
		return ss.enhancer.Close()
	}
	return nil
}

func (ss *ShadowsocksClient) Start() error {
	if err := ss.initEnhancer(); err != nil {
		return err
	}

	n := len(ss.srvs)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		logger.Logger.Infof("start local [%s] server: %s", ss.srvs[i].Type().String(), ss.srvs[i].LocalAddr())
		go ss.srvs[i].Start()
	}

	// check error
	for i := 0; i < n; i++ {
		if err := <-ss.srvs[i].Error(); err != nil {
			return err
		}
	}

	// set system proxy
	if ss.Opts.localOpts.systemProxy {
		ss.setSystemProxy()
	}

	return nil
}

func (ss *ShadowsocksClient) Close() error {
	defer geoip.CloseDB()

	if err := ss.closeTun(); err != nil {
		return err
	}
	n := len(ss.srvs)
	if n == 0 {
		return nil
	}
	for i := 0; i < n-1; i++ {
		ss.srvs[i].Close()
	}
	return ss.srvs[n-1].Close()
}
