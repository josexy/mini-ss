package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/util/logger"
	"gopkg.in/yaml.v3"
)

type TlsOption struct {
	// tls or mtls
	Mode     string `yaml:"mode,omitempty" json:"mode,omitempty"`
	Hostname string `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	KeyPath  string `yaml:"key_path,omitempty" json:"key_path,omitempty"`
	CertPath string `yaml:"cert_path,omitempty" json:"cert_path,omitempty"`
	CAPath   string `yaml:"ca_path,omitempty" json:"ca_path,omitempty"`
}

type WsOption struct {
	Path     string    `yaml:"path" json:"path"`
	Host     string    `yaml:"host,omitempty" json:"host,omitempty"`
	Compress bool      `yaml:"compress,omitempty" json:"compress,omitempty"`
	TLS      TlsOption `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type ObfsOption struct {
	Host string `yaml:"host,omitempty" json:"host,omitempty"`
}

type QuicOption struct {
	Conns                int       `yaml:"conns" json:"conns"`
	KeepAlive            int       `yaml:"keep_alive" json:"keep_alive"`
	MaxIdleTimeout       int       `yaml:"max_idle_timeout" json:"max_idle_timeout"`
	HandshakeIdleTimeout int       `yaml:"handshake_idle_timeout" json:"handshake_idle_timeout"`
	TLS                  TlsOption `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type GrpcOption struct {
	SendBufferSize int       `yaml:"send_buffer_size,omitempty" json:"send_buffer_size,omitempty"`
	RecvBufferSize int       `yaml:"receive_buffer_size,omitempty" json:"receive_buffer_size,omitempty"`
	TLS            TlsOption `yaml:"tls,omitempty" json:"tls,omitempty"`
}

type SshOption struct {
	User          string `yaml:"user" json:"user"`
	Password      string `yaml:"password" json:"password"`
	PrivateKey    string `yaml:"private_key" json:"private_key"`
	PublicKey     string `yaml:"public_key" json:"public_key"`
	AuthorizedKey string `yaml:"authorized_key" json:"authorized_key"`
}

type SSROption struct {
	Protocol      string `yaml:"protocol" json:"protocol"`
	ProtocolParam string `yaml:"protocol_param,omitempty" json:"protocol_param,omitempty"`
	Obfs          string `yaml:"obfs" json:"obfs"`
	ObfsParam     string `yaml:"obfs_param,omitempty" json:"obfs_param,omitempty"`
}

type ServerConfig struct {
	Disable   bool        `yaml:"disable,omitempty" json:"disable,omitempty"`
	Type      string      `yaml:"type,omitempty" json:"type,omitempty"`
	Name      string      `yaml:"name" json:"name"`
	Addr      string      `yaml:"addr" json:"addr"`
	Password  string      `yaml:"password" json:"password"`
	Method    string      `yaml:"method" json:"method"`
	Transport string      `yaml:"transport" json:"transport"`
	Udp       bool        `yaml:"udp,omitempty" json:"udp,omitempty"`
	Ws        *WsOption   `yaml:"ws,omitempty" json:"ws,omitempty"`
	Obfs      *ObfsOption `yaml:"obfs,omitempty" json:"obfs,omitempty"`
	Quic      *QuicOption `yaml:"quic,omitempty" json:"quic,omitempty"`
	Grpc      *GrpcOption `yaml:"grpc,omitempty" json:"grpc,omitempty"`
	Ssh       *SshOption  `yaml:"ssh,omitempty" json:"ssh,omitempty"`
	SSR       *SSROption  `yaml:"ssr,omitempty" json:"ssr,omitempty"`
}

type TunOption struct {
	Enable    bool     `yaml:"enable" json:"enable"`
	Name      string   `yaml:"name" json:"name"`
	Cidr      string   `yaml:"cidr" json:"cidr"`
	Mtu       int      `yaml:"mtu" json:"mtu"`
	DnsHijack []string `yaml:"dns_hijack,omitempty" json:"dns_hijack,omitempty"`
}

type DnsOption struct {
	Listen         string   `yaml:"listen" json:"listen"`
	DomainFilter   []string `yaml:"domain_filter" json:"domain_filter"`
	Nameservers    []string `yaml:"nameservers" json:"nameservers"`
	DisableRewrite bool     `yaml:"disable_rewrite" json:"disable_rewrite"`
}

type MitmFakeCertPool struct {
	Capacity     int `yaml:"capacity" json:"capacity"`
	Interval     int `yaml:"interval" json:"interval"`
	ExpireSecond int `yaml:"expire_second" json:"expire_second"`
}

type MitmOption struct {
	Enable       bool              `yaml:"enable" json:"enable"`
	Proxy        string            `yaml:"proxy" json:"proxy"`
	CAPath       string            `yaml:"ca_path" json:"ca_path"`
	KeyPath      string            `yaml:"key_path" json:"key_path"`
	FakeCertPool *MitmFakeCertPool `yaml:"fake_cert_pool" json:"fake_cert_pool"`
}

type LocalConfig struct {
	SocksAddr   string      `yaml:"socks_addr,omitempty" json:"socks_addr,omitempty"`
	HTTPAddr    string      `yaml:"http_addr,omitempty" json:"http_addr,omitempty"`
	SocksAuth   string      `yaml:"socks_auth,omitempty" json:"socks_auth,omitempty"`
	HTTPAuth    string      `yaml:"http_auth,omitempty" json:"http_auth,omitempty"`
	MixedAddr   string      `yaml:"mixed_addr,omitempty" json:"mixed_addr,omitempty"`
	TCPTunAddr  []string    `yaml:"tcp_tun_addr,omitempty" json:"tcp_tun_addr,omitempty"`
	SystemProxy bool        `yaml:"system_proxy,omitempty" json:"system_proxy,omitempty"`
	Mitm        *MitmOption `yaml:"mitm,omitempty" json:"mitm,omitempty"`
	Tun         *TunOption  `yaml:"tun,omitempty" json:"tun,omitempty"`
	DNS         *DnsOption  `yaml:"dns,omitempty" json:"dns,omitempty"`
}

type Domain struct {
	Proxy  string   `yaml:"proxy" json:"proxy"`
	Action string   `yaml:"action" json:"action"`
	Value  []string `yaml:"value" json:"value"`
}

type DomainKeyword struct {
	Proxy  string   `yaml:"proxy" json:"proxy"`
	Action string   `yaml:"action" json:"action"`
	Value  []string `yaml:"value" json:"value"`
}

type DomainSuffix struct {
	Proxy  string   `yaml:"proxy" json:"proxy"`
	Action string   `yaml:"action" json:"action"`
	Value  []string `yaml:"value" json:"value"`
}

type GeoIP struct {
	Resolve bool     `yaml:"resolve,omitempty" json:"resolve,omitempty"`
	Proxy   string   `yaml:"proxy" json:"proxy"`
	Action  string   `yaml:"action" json:"action"`
	Value   []string `yaml:"value" json:"value"`
}

type IPCidr struct {
	Resolve bool     `yaml:"resolve,omitempty" json:"resolve,omitempty"`
	Proxy   string   `yaml:"proxy" json:"proxy"`
	Action  string   `yaml:"action" json:"action"`
	Value   []string `yaml:"value" json:"value"`
}

type Match struct {
	Others         string           `yaml:"others,omitempty" json:"others,omitempty"`
	Domains        []*Domain        `yaml:"domain,omitempty" json:"domain,omitempty"`
	DomainKeywords []*DomainKeyword `yaml:"domain_keyword,omitempty" json:"domain_keyword,omitempty"`
	DomainSuffixs  []*DomainSuffix  `yaml:"domain_suffix,omitempty" json:"domain_suffix,omitempty"`
	GeoIPs         []*GeoIP         `yaml:"geoip,omitempty" json:"geoip,omitempty"`
	IPCidrs        []*IPCidr        `yaml:"ipcidr,omitempty" json:"ipcidr,omitempty"`
}

type Rules struct {
	Mode     string `yaml:"mode" json:"mode"`
	DirectTo string `yaml:"direct_to,omitempty" json:"direct_to,omitempty"`
	GlobalTo string `yaml:"global_to,omitempty" json:"global_to,omitempty"`
	Match    *Match `yaml:"match,omitempty" json:"match,omitempty"`
}

type LogConfig struct {
	Color        bool   `yaml:"color,omitempty" json:"color,omitempty"`
	LogLevel     string `yaml:"log_level,omitempty" json:"log_level,omitempty"`
	VerboseLevel int    `yaml:"verbose_level,omitempty" json:"verbose_level,omitempty"`
}

type Config struct {
	Server          []*ServerConfig `yaml:"server,omitempty" json:"server,omitempty"`
	Local           *LocalConfig    `yaml:"local,omitempty" json:"local,omitempty"`
	Log             *LogConfig      `yaml:"log,omitempty" json:"log,omitempty"`
	Iface           string          `yaml:"iface,omitempty" json:"iface,omitempty"`
	AutoDetectIface bool            `yaml:"auto_detect_iface,omitempty" json:"auto_detect_iface,omitempty"`
	Rules           *Rules          `yaml:"rules,omitempty" json:"rules,omitempty"`
}

func ParseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	if err = yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (cfg *Config) DeleteServerConfig(name string) {
	index := -1
	for i, c := range cfg.Server {
		if c.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}
	cfg.Server = append(cfg.Server[:index], cfg.Server[index+1:]...)
}

func (cfg *Config) BuildRuler() *rule.Ruler {
	var mode rule.RuleMode
	if cfg.Rules == nil {
		return rule.NewRuler(rule.Direct, "", "", nil)
	}
	switch cfg.Rules.Mode {
	case "global":
		mode = rule.Global
	case "direct":
		mode = rule.Direct
	default:
		mode = rule.Match
	}

	if mode == rule.Global || mode == rule.Direct {
		return rule.NewRuler(mode, cfg.Rules.DirectTo, cfg.Rules.GlobalTo, nil)
	}

	if cfg.Rules.Match == nil {
		logger.Logger.Fatal("the rule mode is match but rules is empty")
	}

	// match mode
	var (
		domainRules        []*rule.RuleItem
		domainKeywordRules []*rule.RuleItem
		domainSuffixRules  []*rule.RuleItem
		geoipRules         []*rule.RuleItem
		ipcidrRules        []*rule.RuleItem
		otherRules         []*rule.RuleItem
	)
	for _, r := range cfg.Rules.Match.Domains {
		domainRules = append(domainRules, &rule.RuleItem{
			RuleMode: rule.Match,
			RuleType: rule.RuleDomain,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range cfg.Rules.Match.DomainKeywords {
		domainKeywordRules = append(domainKeywordRules, &rule.RuleItem{
			RuleMode: rule.Match,
			RuleType: rule.RuleDomainKeyword,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range cfg.Rules.Match.DomainSuffixs {
		domainSuffixRules = append(domainSuffixRules, &rule.RuleItem{
			RuleMode: rule.Match,
			RuleType: rule.RuleDomainSuffix,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range cfg.Rules.Match.GeoIPs {
		geoipRules = append(geoipRules, &rule.RuleItem{
			RuleMode: rule.Match,
			RuleType: rule.RuleGeoIP,
			Resolve:  r.Resolve,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range cfg.Rules.Match.IPCidrs {
		ipcidrRules = append(ipcidrRules, &rule.RuleItem{
			RuleMode: rule.Match,
			RuleType: rule.RuleIPCIDR,
			Resolve:  r.Resolve,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}

	var rules [][]*rule.RuleItem
	otherRules = append(otherRules, &rule.RuleItem{
		RuleMode: rule.Match,
		Proxy:    cfg.Rules.Match.Others,
		RuleType: rule.RuleOthers,
		Accept:   true,
	})
	rules = append(rules, domainRules)
	rules = append(rules, domainKeywordRules)
	rules = append(rules, domainSuffixRules)
	rules = append(rules, geoipRules)
	rules = append(rules, ipcidrRules)
	rules = append(rules, otherRules)
	return rule.NewRuler(mode, cfg.Rules.DirectTo, cfg.Rules.GlobalTo, rules)
}

func (cfg *Config) BuildSSLocalOptions() []ss.SSOption {
	opts := cfg.BuildServerOptions()
	opts = append(opts, cfg.BuildLocalOptions()...)
	return opts
}

func (cfg *Config) BuildServerOptions() []ss.SSOption {
	var res []ss.SSOption

	for _, opt := range cfg.Server {
		if opt.Disable {
			continue
		}
		var opts []ss.SSOption

		switch opt.Transport {
		case "ws":
			opts = append(opts, ss.WithWsTransport())
			opts = append(opts, ss.WithWsHost(opt.Ws.Host))
			opts = append(opts, ss.WithWsPath(opt.Ws.Path))
			opts = append(opts, ss.WithWsCertPath(opt.Ws.TLS.CertPath))
			opts = append(opts, ss.WithWsKeyPath(opt.Ws.TLS.KeyPath))
			opts = append(opts, ss.WithWsCAPath(opt.Ws.TLS.CAPath))
			opts = append(opts, ss.WithWsHostname(opt.Ws.TLS.Hostname))
			if opt.Ws.Compress {
				opts = append(opts, ss.WithWsCompress())
			}
			switch opt.Ws.TLS.Mode {
			case "tls":
				opts = append(opts, ss.WithWsTLS(options.TLS))
			case "mtls":
				opts = append(opts, ss.WithWsTLS(options.MTLS))
			}
		case "obfs":
			opts = append(opts, ss.WithObfsTransport())
			opts = append(opts, ss.WithObfsHost(opt.Obfs.Host))
		case "quic":
			opts = append(opts, ss.WithQuicTransport())
			opts = append(opts, ss.WithQuicConns(opt.Quic.Conns))
			opts = append(opts, ss.WithQuicKeepAlivePeriod(time.Second*time.Duration(opt.Quic.KeepAlive)))
			opts = append(opts, ss.WithQuicMaxIdleTimeout(time.Second*time.Duration(opt.Quic.MaxIdleTimeout)))
			opts = append(opts, ss.WithQuicHandshakeIdleTimeout(time.Second*time.Duration(opt.Quic.HandshakeIdleTimeout)))
			opts = append(opts, ss.WithQuicCertPath(opt.Quic.TLS.CertPath))
			opts = append(opts, ss.WithQuicKeyPath(opt.Quic.TLS.KeyPath))
			opts = append(opts, ss.WithQuicCAPath(opt.Quic.TLS.CAPath))
			opts = append(opts, ss.WithQuicHostname(opt.Quic.TLS.Hostname))
			switch opt.Quic.TLS.Mode {
			case "tls":
				opts = append(opts, ss.WithQuicTLS(options.TLS))
			case "mtls":
				opts = append(opts, ss.WithQuicTLS(options.MTLS))
			}
		case "grpc":
			opts = append(opts, ss.WithGrpcTransport())
			opts = append(opts, ss.WithGrpcSndRevBuffer(opt.Grpc.SendBufferSize, opt.Grpc.RecvBufferSize))
			opts = append(opts, ss.WithGrpcCertPath(opt.Grpc.TLS.CertPath))
			opts = append(opts, ss.WithGrpcKeyPath(opt.Grpc.TLS.KeyPath))
			opts = append(opts, ss.WithGrpcCAPath(opt.Grpc.TLS.CAPath))
			opts = append(opts, ss.WithGrpcHostname(opt.Grpc.TLS.Hostname))
			switch opt.Grpc.TLS.Mode {
			case "tls":
				opts = append(opts, ss.WithGrpcTLS(options.TLS))
			case "mtls":
				opts = append(opts, ss.WithGrpcTLS(options.MTLS))
			}
		case "ssh":
			opts = append(opts, ss.WithSshTransport())
			opts = append(opts, ss.WithSshUser(opt.Ssh.User))
			opts = append(opts, ss.WithSshPassword(opt.Ssh.Password))
			opts = append(opts, ss.WithSshPrivateKey(opt.Ssh.PrivateKey))
			opts = append(opts, ss.WithSshPublicKey(opt.Ssh.PublicKey))
			opts = append(opts, ss.WithSshAuthorizedKey(opt.Ssh.AuthorizedKey))
		case "default":
			// whether to support ssr
			if opt.Type == "ssr" {
				opts = append(opts, ss.WithEnableSSR())
				opts = append(opts, ss.WithSSRProtocol(opt.SSR.Protocol))
				opts = append(opts, ss.WithSSRProtocolParam(opt.SSR.ProtocolParam))
				opts = append(opts, ss.WithSSRObfs(opt.SSR.Obfs))
				opts = append(opts, ss.WithObfsParam(opt.SSR.ObfsParam))
			}
		}

		// default name
		opts = append(opts, ss.WithServerName(opt.Name))
		opts = append(opts, ss.WithServerAddr(opt.Addr))
		opts = append(opts, ss.WithMethod(opt.Method))
		opts = append(opts, ss.WithPassword(opt.Password))
		opts = append(opts, ss.WithUDPRelay(opt.Udp))

		res = append(res, ss.WithServerCompose(opts...))
	}

	// outbound interface
	res = append(res, ss.WithOutboundInterface(cfg.Iface))
	// auto detect interface
	res = append(res, ss.WithAutoDetectInterface(cfg.AutoDetectIface))
	return res
}

func (cfg *Config) BuildLocalOptions() []ss.SSOption {
	var opts []ss.SSOption

	if cfg.Local.MixedAddr != "" {
		opts = append(opts, ss.WithMixedAddr(cfg.Local.MixedAddr))
		opts = append(opts, ss.WithHttpUserInfo(splitAuthInfo(cfg.Local.HTTPAuth)))
		opts = append(opts, ss.WithSocksUserInfo(splitAuthInfo(cfg.Local.SocksAuth)))
	} else {
		if cfg.Local.SocksAddr != "" {
			opts = append(opts, ss.WithSocksAddr(cfg.Local.SocksAddr))
			opts = append(opts, ss.WithSocksUserInfo(splitAuthInfo(cfg.Local.SocksAuth)))
		}
		if cfg.Local.HTTPAddr != "" {
			opts = append(opts, ss.WithHttpAddr(cfg.Local.HTTPAddr))
			opts = append(opts, ss.WithHttpUserInfo(splitAuthInfo(cfg.Local.HTTPAuth)))
		}
	}

	// simple tcp tun address
	var tcpTunAddr [][]string
	for _, addr := range cfg.Local.TCPTunAddr {
		lr, _ := splitTunAddrInfo(addr)
		tcpTunAddr = append(tcpTunAddr, lr)
	}
	opts = append(opts, ss.WithTcpTunAddr(tcpTunAddr))
	opts = append(opts, ss.WithRuler(cfg.BuildRuler()))

	if cfg.Local.Mitm != nil && cfg.Local.Mitm.Enable {
		opts = append(opts, ss.WithMitm(cfg.Local.Mitm.Enable))
		opts = append(opts, ss.WithMitmProxy(cfg.Local.Mitm.Proxy))
		opts = append(opts, ss.WithMitmCAPath(cfg.Local.Mitm.CAPath))
		opts = append(opts, ss.WithMitmKeyPath(cfg.Local.Mitm.KeyPath))
		if cfg.Local.Mitm.FakeCertPool != nil {
			opts = append(opts, ss.WithMitmFakeCertPool(
				cfg.Local.Mitm.FakeCertPool.Capacity,
				cfg.Local.Mitm.FakeCertPool.Interval,
				cfg.Local.Mitm.FakeCertPool.ExpireSecond,
			))
		}
	}
	if cfg.Local.Tun != nil && cfg.Local.Tun.Enable {
		if cfg.Local.DNS == nil {
			logger.Logger.Fatal("if tun mode is enabled, the fake dns configuration must be configured")
		}
		opts = append(opts, ss.WithEnableTun())
		opts = append(opts, ss.WithTunName(cfg.Local.Tun.Name))
		opts = append(opts, ss.WithTunCIDR(cfg.Local.Tun.Cidr))
		opts = append(opts, ss.WithTunMTU(uint32(cfg.Local.Tun.Mtu)))
		opts = append(opts, ss.WithTunDnsHijack(cfg.Local.Tun.DnsHijack))
	}
	if cfg.Local.DNS != nil {
		opts = append(opts, ss.WithFakeDnsServer(cfg.Local.DNS.Listen))
		opts = append(opts, ss.WithFakeDnsDisableRewrite(cfg.Local.DNS.DisableRewrite))
		opts = append(opts, ss.WithFakeDnsDomainFilter(cfg.Local.DNS.DomainFilter))
		opts = append(opts, ss.WithDefaultDnsNameservers(cfg.Local.DNS.Nameservers))
	}

	if cfg.Local.SystemProxy {
		opts = append(opts, ss.WithSystemProxy())
	}

	return opts
}

func splitAuthInfo(auth string) (username, password string) {
	username, password, _ = strings.Cut(auth, ":")
	return
}

func splitTunAddrInfo(addr string) ([]string, error) {
	local, remote, found := strings.Cut(addr, "=")
	if !found {
		return nil, fmt.Errorf("tun address invalid: %q", addr)
	}
	return []string{local, remote}, nil
}
