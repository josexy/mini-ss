package ss

import (
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/josexy/mini-ss/enhancer"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/proxy"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/ssr"
	"github.com/josexy/mini-ss/transport"
)

type serverOptions struct {
	name      string
	addr      string
	method    string
	password  string
	transport transport.Type
	udp       bool
	opts      options.Options
	ssr       bool
	ssrOpt    ssr.ShadowsocksROption
}

type localOptions struct {
	socksAddr      string
	httpAddr       string
	mixedAddr      string
	socksAuth      *Auth
	httpAuth       *Auth
	tcpTunAddr     [][]string
	systemProxy    bool
	enableTun      bool
	enhancerConfig enhancer.EnhancerConfig
	mitmConfig     proxy.MimtOption
}

type ssOptions struct {
	serverOpts []serverOptions
	localOpts  localOptions
}

type SSOption interface{ applyTo(*ssOptions) }

type ssOptionFunc func(*ssOptions)

func (f ssOptionFunc) applyTo(o *ssOptions) { f(o) }

// WithEnableTun enable tun mode (client-only)
func WithEnableTun() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enableTun = true
	})
}

func WithTunName(name string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enhancerConfig.Tun.Name = name
	})
}

func WithTunCIDR(cidr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enhancerConfig.Tun.Addr = cidr
	})
}

func WithTunMTU(mtu uint32) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enhancerConfig.Tun.MTU = mtu
	})
}

func WithTunDnsHijack(dns []string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		for _, addr := range dns {
			if strings.Contains(addr, "any") {
				addr = strings.ReplaceAll(addr, "any", "0.0.0.0")
			}
			host, port, _ := net.SplitHostPort(addr)
			if host == "" || port == "" {
				port = "53"
				addr = net.JoinHostPort(addr, port)
			}
			if port != "53" {
				continue
			}
			addrPort, err := netip.ParseAddrPort(addr)
			if err != nil || !addrPort.IsValid() {
				continue
			}
			so.localOpts.enhancerConfig.DnsHijack = append(so.localOpts.enhancerConfig.DnsHijack, addrPort)
		}
	})
}

func WithFakeDnsDomainFilter(domain []string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		resolver.DefaultDomainFilter = domain
	})
}

func WithFakeDnsServer(addr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enhancerConfig.FakeDNS = addr
	})
}

func WithFakeDnsDisableRewrite(disable bool) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enhancerConfig.DisableRewrite = disable
	})
}

// WithOutboundInterface set the outgoing interface name
func WithOutboundInterface(ifaceName string) SSOption {
	return ssOptionFunc(func(*ssOptions) {
		options.DefaultOptions.OutboundInterface = ifaceName
	})
}

// WithDefaultDnsNameservers default dns nameservers
func WithDefaultDnsNameservers(ns []string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		if len(ns) == 0 {
			return
		}
		resolver.DefaultDnsNameservers = ns
	})
}

func WithAutoDetectInterface(enable bool) SSOption {
	return ssOptionFunc(func(*ssOptions) {
		options.DefaultOptions.AutoDetectInterface = enable
	})
}

func WithServerCompose(opts ...SSOption) SSOption {
	o := ssOptions{serverOpts: make([]serverOptions, 1)}
	for _, opt := range opts {
		opt.applyTo(&o)
	}
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts = append(so.serverOpts, o.serverOpts...)
	})
}

func WithServerName(name string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].name = name
	})
}

func WithServerAddr(addr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].addr = addr
	})
}

func WithUDPRelay(enable bool) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].udp = enable
	})
}

// WithEnableSSR whether to support SSR connection
// for example "ss" or "ssr", default "ss"
func WithEnableSSR() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].ssr = true
	})
}

// WithSSRProtocol ssr protocol name
// support protocol plugins:
// - origin
// - auth_sha1_v4
// - auth_aes128_md5
// - auth_aes128_sha1
// - auth_chain_a
// - auth_chain_b
func WithSSRProtocol(name string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].ssrOpt.Protocol = name
	})
}

func WithSSRProtocolParam(param string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].ssrOpt.ProtocolParam = param
	})
}

// WithSSRObfs ssr obfs name
// support obfs plugins:
// - plain
// - http_simple
// - http_post
// - random_head
// - tls1.2_ticket_auth
// - tls1.2_ticket_auth_compatible
func WithSSRObfs(name string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].ssrOpt.Obfs = name
	})
}

func WithObfsParam(param string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].ssrOpt.ObfsParam = param
	})
}

func WithMethod(method string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].method = method
	})
}

func WithPassword(password string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].password = password
	})
}

func WithDefaultTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Tcp
		clone := *options.DefaultOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithRuler(ruler *rule.Ruler) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		rule.MatchRuler = ruler
	})
}

// WithSystemProxy whether to enable system proxy (for Linux, only Ubuntu and KDE are supported)
func WithSystemProxy() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.systemProxy = true
	})
}

func WithSocksUserInfo(username, password string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.socksAuth = NewAuth(username, password)
	})
}

func WithSocksAddr(addr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.socksAddr = addr
	})
}

func WithHttpAddr(addr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.httpAddr = addr
	})
}

func WithHttpUserInfo(username, password string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.httpAuth = NewAuth(username, password)
	})
}

// WithMixedAddr mixed proxy ports (SOCKS and HTTP)
func WithMixedAddr(addr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.mixedAddr = addr
	})
}

func WithTcpTunAddr(addrs [][]string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.tcpTunAddr = addrs
	})
}

func WithMitm(enable bool) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.mitmConfig.Enable = enable
	})
}

func WithMitmCAPath(caPath string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.mitmConfig.CaPath = caPath
	})
}

func WithMitmKeyPath(keyPath string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.mitmConfig.KeyPath = keyPath
	})
}

func WithMitmFakeCertPool(capacity, interval, expireSecond int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.mitmConfig.FakeCertPool.Capacity = capacity
		so.localOpts.mitmConfig.FakeCertPool.Interval = interval
		so.localOpts.mitmConfig.FakeCertPool.ExpireSecond = expireSecond
	})
}

func WithObfsTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Obfs
		clone := *options.DefaultObfsOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithObfsHost(host string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.ObfsOptions).Host = host
	})
}

func WithKcpTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Kcp
		clone := *options.DefaultKcpOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithKcpKey(key string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Key = key
	})
}

// WithKcpCrypt support: none, aes-128, aes-192, salsa20, blowfish, twofish, cast5, 3des, tea, xtea, xor, sm4
func WithKcpCrypt(crypt string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Crypt = crypt
	})
}

// WithKcpMode fast3, fast2, fast, normal, manual
func WithKcpMode(mode string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Mode = mode
	})
}

func WithKcpMTU(mtu int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Mtu = mtu
	})
}

func WithKcpSndRevWnd(sndWnd, revWnd int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).SndWnd = sndWnd
		so.serverOpts[0].opts.(*options.KcpOptions).RevWnd = revWnd
	})
}

func WithKcpDataShard(dataShard int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).DataShard = dataShard
	})
}

func WithKcpParityShard(parityShard int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).ParityShard = parityShard
	})
}

func WithKcpDscp(dscp int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Dscp = dscp
	})
}

func WithKcpCompress() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).NoCompress = false
	})
}

func WithKcpAckNoDelay() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).AckNoDelay = true
	})
}

// WithKcpNoDelay
// NoDelay options
// fastest: ikcp_nodelay(kcp, 1, 20, 2, 1)
// nodelay: 0:disable(default), 1:enable
// interval: internal update timer interval in millisec, default is 100ms
// resend: 0:disable fast resend(default), 1:enable fast resend
// nc: 0:normal congestion control(default), 1:disable congestion control
func WithKcpNoDelay(noDelay int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).NoDelay = noDelay
	})
}

func WithKcpInterval(interval int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Interval = interval
	})
}

func WithKcpResend(resend int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Resend = resend
	})
}

func WithKcpNc(nc int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).Nc = nc
	})
}

func WithKcpSockBuf(sockBuf int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).SockBuf = sockBuf
	})
}

func WithKcpSmuxVer(smuxVer int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).SmuxVer = smuxVer
	})
}

func WithKcpSmuxBuf(smuxBuf int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).SmuxBuf = smuxBuf
	})
}

func WithKcpStreamBuf(streamBuf int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).StreamBuf = streamBuf
	})
}

func WithKcpKeepAlive(second int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.KcpOptions).KeepAlive = second
	})
}

func WithKcpConns(conns int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		if conns <= 0 {
			return
		}
		so.serverOpts[0].opts.(*options.KcpOptions).Conns = conns
	})
}

func WithQuicTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Quic
		clone := *options.DefaultQuicOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithQuicHandshakeIdleTimeout(timeout time.Duration) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).HandshakeIdleTimeout = timeout
	})
}

func WithQuicKeepAlivePeriod(timeout time.Duration) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).KeepAlivePeriod = timeout
	})
}

func WithQuicMaxIdleTimeout(timeout time.Duration) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).MaxIdleTimeout = timeout
	})
}

func WithQuicConns(conns int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		if conns <= 0 {
			return
		}
		so.serverOpts[0].opts.(*options.QuicOptions).Conns = conns
	})
}

func WithQuicTLS(mode options.TlsMode) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).TlsOptions.Mode = mode
	})
}

func WithQuicHostname(hostname string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).TlsOptions.Hostname = hostname
	})
}

func WithQuicCertPath(certFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).TlsOptions.CertFile = certFile
	})
}

func WithQuicKeyPath(keyFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).TlsOptions.KeyFile = keyFile
	})
}

func WithQuicCAPath(caFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.QuicOptions).TlsOptions.CAFile = caFile
	})
}

func WithWsTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Websocket
		clone := *options.DefaultWsOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithWsHost(host string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).Host = host
	})
}

func WithWsPath(path string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).Path = path
	})
}

func WithWsSndRevBuffer(sndBuffer, revBuffer int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).SndBuffer = sndBuffer
		so.serverOpts[0].opts.(*options.WsOptions).RevBuffer = revBuffer
	})
}

func WithWsCompress() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).Compress = true
	})
}

func WithWsUserAgent(userAgent string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).UserAgent = userAgent
	})
}

func WithWsTLS(mode options.TlsMode) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).TlsOptions.Mode = mode
	})
}

func WithWsHostname(hostname string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).TlsOptions.Hostname = hostname
	})
}

func WithWsCertPath(certFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).TlsOptions.CertFile = certFile
	})
}

func WithWsKeyPath(keyFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).TlsOptions.KeyFile = keyFile
	})
}

func WithWsCAPath(caFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.WsOptions).TlsOptions.CAFile = caFile
	})
}

func WithGrpcTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Grpc
		clone := *options.DefaultGrpcOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithGrpcTLS(mode options.TlsMode) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.GrpcOptions).TlsOptions.Mode = mode
	})
}

func WithGrpcHostname(hostname string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.GrpcOptions).Hostname = hostname
	})
}

func WithGrpcCertPath(certFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.GrpcOptions).TlsOptions.CertFile = certFile
	})
}

func WithGrpcKeyPath(keyFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.GrpcOptions).TlsOptions.KeyFile = keyFile
	})
}

func WithGrpcCAPath(caFile string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.GrpcOptions).TlsOptions.CAFile = caFile
	})
}

func WithGrpcSndRevBuffer(sndBuffer, revBuffer int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*options.GrpcOptions).SndBuffer = sndBuffer
		so.serverOpts[0].opts.(*options.GrpcOptions).RevBuffer = revBuffer
	})
}
