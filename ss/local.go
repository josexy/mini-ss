package ss

import (
	"context"
	"net"
	"net/url"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/server"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/ssr"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/tun"
	"github.com/josexy/mini-ss/util"
	"github.com/josexy/mini-ss/util/proxyutil"
)

var defaultSSLocalOpts = ssOptions{
	localOpts: localOptions{
		tunCfg: tun.TunConfig{
			Name: "utun3",
			Addr: "198.18.0.1/16",
			MTU:  uint32(tun.DefaultMTU),
		},
	},
}

type ShadowsocksClient struct {
	srvs []server.Server
	Opts ssOptions
	// tun mode
	tunStack *tun.TunStack
}

func NewShadowsocksClient(opts ...SSOption) *ShadowsocksClient {
	s := &ShadowsocksClient{
		Opts: defaultSSLocalOpts,
	}
	for _, o := range opts {
		o.applyTo(&s.Opts)
	}

	// is support auto-detect-interface
	if transport.DefaultDialerOutboundOption.AutoDetectInterface {
		if ifaceName, err := util.ResolveDefaultRouteInterface(); err == nil {
			transport.DefaultDialerOutboundOption.Interface = ifaceName
			logx.Info("auto detect outbound interface: %q", ifaceName)
		}
	}

	// init global default dns resolver
	resolver.DefaultResolver = resolver.NewDnsResolver(dns.DefaultDnsNameservers)
	cpv := &ctxv.ContextPassValue{R: s.Opts.localOpts.ruler}

	for _, opt := range s.Opts.serverOpts {
		sc, ac, err := cipher.GetCipher(opt.method, opt.password)
		if err != nil {
			logx.FatalBy(err)
		}

		var tcpBound transport.TcpConnBound
		var udpBound transport.UdpConnBound

		// the server whether support shadowsocksr
		if !opt.ssr {
			tcpBound = makeStreamConn(sc, ac)
			udpBound = makePacketConn(sc, ac)
		} else {
			cp, err := ssr.NewSSRClientStreamCipher(sc,
				opt.addr,                                      // host,port
				opt.ssrOpt.Protocol, opt.ssrOpt.ProtocolParam, // protocol,protocol-param
				opt.ssrOpt.Obfs, opt.ssrOpt.ObfsParam) // obfs,obfs-param

			if err != nil {
				logx.FatalBy(err)
			}

			tcpBound = makeSSRClientStreamConn(cp)
			udpBound = makeSSRClientPacketConn(cp)
		}

		cpv.Set(opt.name, ctxv.V{
			Addr:         opt.addr,
			Options:      opt.opts,
			Type:         opt.transport,
			TcpConnBound: tcpBound,
			UdpConnBound: udpBound,
		})
	}
	// ss-local context value
	ctx := context.WithValue(context.Background(), ctxv.SSLocalContextKey, cpv)
	// create tcp tun server
	for _, addrs := range s.Opts.localOpts.tcpTunAddr {
		s.srvs = append(s.srvs, newTcpTunServer(ctx, addrs[0], addrs[1]).Build())
	}

	// create udp tun server
	for _, addrs := range s.Opts.localOpts.udpTunAddr {
		s.srvs = append(s.srvs, newUdpTunServer(ctx, addrs[0], addrs[1]).Build())
	}
	// enable mixed proxy
	if s.Opts.localOpts.mixedAddr != "" {
		s.srvs = append(s.srvs, newMixedServer(ctx, s.Opts.localOpts.mixedAddr).Build())
	} else {
		if s.Opts.localOpts.httpAddr != "" {
			// http proxy
			s.srvs = append(s.srvs, newHttpProxyServer(ctx, s.Opts.localOpts.httpAddr, s.Opts.localOpts.httpAuth).Build())
		}
		if s.Opts.localOpts.socksAddr != "" {
			// socks proxy
			s.srvs = append(s.srvs, newSocksProxyServer(ctx, s.Opts.localOpts.socksAddr, s.Opts.localOpts.socksAuth).Build())
		}
	}
	// userspace netstack gVisor
	if s.Opts.localOpts.enableTun {
		s.tunStack = tun.NewTunStack(ctx, s.Opts.localOpts.tunCfg)
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

func (ss *ShadowsocksClient) initTun() error {
	if ss.Opts.localOpts.enableTun {
		return ss.tunStack.Start()
	}
	return nil
}

func (ss *ShadowsocksClient) closeTun() error {
	if ss.Opts.localOpts.enableTun {
		return ss.tunStack.Close()
	}
	return nil
}

func (ss *ShadowsocksClient) Start() error {
	// open geoip database
	if err := geoip.OpenDB(); err != nil {
		return err
	}

	// init global statistic manager
	statistic.InitGlobalStatisticManager()

	if err := ss.initTun(); err != nil {
		return err
	}

	ss.srvs = removeNilServer(ss.srvs)

	n := len(ss.srvs)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		logx.Info("start local [%s] server: %s", ss.srvs[i].Type().String(), ss.srvs[i].LocalAddr())
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

func removeNilServer(srvs []server.Server) []server.Server {
	var j int
	for i := 0; i < len(srvs); i++ {
		if srvs[i] != nil {
			srvs[j] = srvs[i]
			j++
		}
	}
	return srvs[:j]
}
