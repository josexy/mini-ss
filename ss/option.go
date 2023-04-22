package ss

import (
	"time"

	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/enhancer"
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
	opts      transport.Options
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

func WithFakeDnsServer(addr string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.localOpts.enhancerConfig.FakeDNS = addr
	})
}

// WithOutboundInterface set the outgoing interface name
func WithOutboundInterface(ifaceName string) SSOption {
	return ssOptionFunc(func(*ssOptions) {
		transport.DefaultDialerOutboundOption.Interface = ifaceName
	})
}

// WithDefaultDnsNameservers default dns nameservers
func WithDefaultDnsNameservers(ns []string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		if len(ns) == 0 {
			return
		}
		dns.DefaultDnsNameservers = ns
	})
}

func WithAutoDetectInterface(enable bool) SSOption {
	return ssOptionFunc(func(*ssOptions) {
		transport.DefaultDialerOutboundOption.AutoDetectInterface = enable
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

func WithObfsTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Obfs
		clone := *transport.DefaultObfsOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithObfsHost(host string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.ObfsOptions).Host = host
	})
}

func WithKcpTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Kcp
		clone := *transport.DefaultKcpOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithKcpKey(key string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Key = key
	})
}

// WithKcpCrypt support: none, aes-128, aes-192, salsa20, blowfish, twofish, cast5, 3des, tea, xtea, xor, sm4
func WithKcpCrypt(crypt string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Crypt = crypt
	})
}

// WithKcpMode fast3, fast2, fast, normal, manual
func WithKcpMode(mode string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Mode = mode
	})
}

func WithKcpMTU(mtu int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Mtu = mtu
	})
}

func WithKcpSndRevWnd(sndWnd, revWnd int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).SndWnd = sndWnd
		so.serverOpts[0].opts.(*transport.KcpOptions).RevWnd = revWnd
	})
}

func WithKcpDataShard(dataShard int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).DataShard = dataShard
	})
}

func WithKcpParityShard(parityShard int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).ParityShard = parityShard
	})
}

func WithKcpDscp(dscp int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Dscp = dscp
	})
}

func WithKcpCompress() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).NoCompress = false
	})
}

func WithKcpAckNoDelay() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).AckNoDelay = true
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
		so.serverOpts[0].opts.(*transport.KcpOptions).NoDelay = noDelay
	})
}

func WithKcpInterval(interval int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Interval = interval
	})
}

func WithKcpResend(resend int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Resend = resend
	})
}

func WithKcpNc(nc int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Nc = nc
	})
}

func WithKcpSockBuf(sockBuf int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).SockBuf = sockBuf
	})
}

func WithKcpSmuxVer(smuxVer int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).SmuxVer = smuxVer
	})
}

func WithKcpSmuxBuf(smuxBuf int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).SmuxBuf = smuxBuf
	})
}

func WithKcpStreamBuf(streamBuf int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).StreamBuf = streamBuf
	})
}

func WithKcpKeepAlive(second int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).KeepAlive = second
	})
}

func WithKcpConns(conns int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.KcpOptions).Conns = conns
	})
}

func WithQuicTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Quic
		clone := *transport.DefaultQuicOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithQuicHandshakeIdleTimeout(timeout time.Duration) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.QuicOptions).HandshakeIdleTimeout = timeout
	})
}

func WithQuicKeepAlivePeriod(timeout time.Duration) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.QuicOptions).KeepAlivePeriod = timeout
	})
}

func WithQuicMaxIdleTimeout(timeout time.Duration) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.QuicOptions).MaxIdleTimeout = timeout
	})
}

func WithQuicConns(conns int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.QuicOptions).Conns = conns
	})
}

func WithWsTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Websocket
		clone := *transport.DefaultWsOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithWsHost(host string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.WsOptions).Host = host
	})
}

func WithWsPath(path string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.WsOptions).Path = path
	})
}

func WithWsSndRevBuffer(sndBuffer, revBuffer int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.WsOptions).SndBuffer = sndBuffer
		so.serverOpts[0].opts.(*transport.WsOptions).RevBuffer = revBuffer
	})
}

func WithWsCompress() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.WsOptions).Compress = true
	})
}

func WithWsUserAgent(userAgent string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.WsOptions).UserAgent = userAgent
	})
}

func WithWsTLS() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.WsOptions).TLS = true
	})
}

func WithGrpcTransport() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].transport = transport.Grpc
		clone := *transport.DefaultGrpcOptions
		so.serverOpts[0].opts = &clone
	})
}

func WithGrpcHostname(hostname string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.GrpcOptions).Hostname = hostname
	})
}

func WithGrpcTLS() SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.GrpcOptions).TLS = true
	})
}

func WithGrpcCertPath(certPath string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.GrpcOptions).CertPath = certPath
	})
}

func WithGrpcKeyPath(keyPath string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.GrpcOptions).KeyPath = keyPath
	})
}

func WithGrpcCAPath(caPath string) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.GrpcOptions).CAPath = caPath
	})
}

func WithGrpcSndRevBuffer(sndBuffer, revBuffer int) SSOption {
	return ssOptionFunc(func(so *ssOptions) {
		so.serverOpts[0].opts.(*transport.GrpcOptions).SndBuffer = sndBuffer
		so.serverOpts[0].opts.(*transport.GrpcOptions).RevBuffer = revBuffer
	})
}
