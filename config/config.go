package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/util/logger"
	"gopkg.in/yaml.v3"
)

type KcpOption struct {
	Crypt    string `yaml:"crypt"`
	Key      string `yaml:"key"`
	Mode     string `yaml:"mode"`
	Compress bool   `yaml:"compress"`
	Conns    int    `yaml:"conns"`
}

type WsOption struct {
	Host     string `yaml:"host"`
	Path     string `yaml:"path"`
	Compress bool   `yaml:"compress"`
	TLS      bool   `yaml:"tls"`
}

type ObfsOption struct {
	Host string `yaml:"host"`
}

type QuicOption struct {
	Conns int `yaml:"conns"`
}

type SSROption struct {
	Protocol      string `yaml:"protocol"`
	ProtocolParam string `yaml:"protocol_param"`
	Obfs          string `yaml:"obfs"`
	ObfsParam     string `yaml:"obfs_param"`
}

type ServerConfig struct {
	Disable   bool        `yaml:"disable"`
	Type      string      `yaml:"type,omitempty"`
	Name      string      `yaml:"name"`
	Addr      string      `yaml:"addr"`
	Password  string      `yaml:"password"`
	Method    string      `yaml:"method"`
	Transport string      `yaml:"transport"`
	Udp       bool        `yaml:"udp"`
	Kcp       *KcpOption  `yaml:"kcp,omitempty"`
	Ws        *WsOption   `yaml:"ws,omitempty"`
	Obfs      *ObfsOption `yaml:"obfs,omitempty"`
	Quic      *QuicOption `yaml:"quic,omitempty"`
	SSR       *SSROption  `yaml:"ssr,omitempty"`
}

type TunOption struct {
	Name string `yaml:"name"`
	Cidr string `yaml:"cidr"`
	Mtu  int    `yaml:"mtu"`
}

type FakeDnsOption struct {
	Listen      string   `yaml:"listen"`
	Nameservers []string `yaml:"nameservers"`
}

type LocalConfig struct {
	SocksAddr   string         `yaml:"socks_addr"`
	HTTPAddr    string         `yaml:"http_addr"`
	SocksAuth   string         `yaml:"socks_auth"`
	HTTPAuth    string         `yaml:"http_auth"`
	MixedAddr   string         `yaml:"mixed_addr"`
	TCPTunAddr  []string       `yaml:"tcp_tun_addr"`
	SystemProxy bool           `yaml:"system_proxy"`
	
	EnableTun   bool           `yaml:"enable_tun"`
	Tun         *TunOption     `yaml:"tun,omitempty"`
	FakeDNS     *FakeDnsOption `yaml:"fake_dns,omitempty"`
}

type Domain struct {
	Proxy  string   `yaml:"proxy"`
	Action string   `yaml:"action"`
	Value  []string `yaml:"value"`
}

type DomainKeyword struct {
	Proxy  string   `yaml:"proxy"`
	Action string   `yaml:"action"`
	Value  []string `yaml:"value"`
}

type DomainSuffix struct {
	Proxy  string   `yaml:"proxy"`
	Action string   `yaml:"action"`
	Value  []string `yaml:"value"`
}

type GeoIP struct {
	Resolve bool     `yaml:"resolve"`
	Proxy   string   `yaml:"proxy"`
	Action  string   `yaml:"action"`
	Value   []string `yaml:"value"`
}

type IPCidr struct {
	Resolve bool     `yaml:"resolve"`
	Proxy   string   `yaml:"proxy"`
	Action  string   `yaml:"action"`
	Value   []string `yaml:"value"`
}

type Match struct {
	Others         string           `yaml:"others,omitempty"`
	Domains        []*Domain        `yaml:"domain,omitempty"`
	DomainKeywords []*DomainKeyword `yaml:"domain_keyword,omitempty"`
	DomainSuffixs  []*DomainSuffix  `yaml:"domain_suffix,omitempty"`
	GeoIPs         []*GeoIP         `yaml:"geoip,omitempty"`
	IPCidrs        []*IPCidr        `yaml:"ipcidr,omitempty"`
}

type Rules struct {
	Mode     string `yaml:"mode"`
	DirectTo string `yaml:"direct_to"`
	GlobalTo string `yaml:"global_to"`
	Match    *Match `yaml:"match,omitempty"`
}

type Config struct {
	Server          []*ServerConfig `yaml:"server,omitempty"`
	Local           *LocalConfig    `yaml:"local,omitempty"`
	Color           bool            `yaml:"color"`
	Verbose         bool            `yaml:"verbose"`
	VerboseLevel    int             `yaml:"verbose_level"`
	Iface           string          `yaml:"iface"`
	AutoDetectIface bool            `yaml:"auto_detect_iface"`
	Rules           *Rules          `yaml:"rules"`
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
		case "kcp":
			opts = append(opts, ss.WithKcpTransport())
			opts = append(opts, ss.WithKcpCrypt(opt.Kcp.Crypt))
			opts = append(opts, ss.WithKcpKey(opt.Kcp.Key))
			opts = append(opts, ss.WithKcpMode(opt.Kcp.Mode))
			opts = append(opts, ss.WithKcpConns(opt.Kcp.Conns))
			if opt.Kcp.Compress {
				opts = append(opts, ss.WithKcpCompress())
			}
		case "ws":
			opts = append(opts, ss.WithWsTransport())
			opts = append(opts, ss.WithWsHost(opt.Ws.Host))
			opts = append(opts, ss.WithWsPath(opt.Ws.Path))
			if opt.Ws.Compress {
				opts = append(opts, ss.WithWsCompress())
			}
			if opt.Ws.TLS {
				opts = append(opts, ss.WithWsTLS())
			}
		case "obfs":
			opts = append(opts, ss.WithObfsTransport())
			opts = append(opts, ss.WithObfsHost(opt.Obfs.Host))
		case "quic":
			opts = append(opts, ss.WithQuicTransport())
			opts = append(opts, ss.WithQuicConns(opt.Quic.Conns))
		case "default":
			opts = append(opts, ss.WithUDPRelay(opt.Udp))
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

	if cfg.Local.SocksAddr != "" {
		opts = append(opts, ss.WithSocksAddr(cfg.Local.SocksAddr))
		opts = append(opts, ss.WithSocksUserInfo(splitAuthInfo(cfg.Local.SocksAuth)))
	}

	if cfg.Local.HTTPAddr != "" {
		opts = append(opts, ss.WithHttpAddr(cfg.Local.HTTPAddr))
		opts = append(opts, ss.WithHttpUserInfo(splitAuthInfo(cfg.Local.HTTPAuth)))
	}

	opts = append(opts, ss.WithMixedAddr(cfg.Local.MixedAddr))

	// simple tcp tun address
	var tcpTunAddr [][]string
	for _, addr := range cfg.Local.TCPTunAddr {
		lr, _ := splitTunAddrInfo(addr)
		tcpTunAddr = append(tcpTunAddr, lr)
	}
	opts = append(opts, ss.WithTcpTunAddr(tcpTunAddr))

	opts = append(opts, ss.WithRuler(cfg.BuildRuler()))

	if cfg.Local.EnableTun {
		if cfg.Local.FakeDNS == nil {
			logger.Logger.Fatal("if tun mode is enabled, the fake dns configuration must exist")
		}
		opts = append(opts, ss.WithEnableTun())
		opts = append(opts, ss.WithTunName(cfg.Local.Tun.Name))
		opts = append(opts, ss.WithTunCIDR(cfg.Local.Tun.Cidr))
		opts = append(opts, ss.WithTunMTU(uint32(cfg.Local.Tun.Mtu)))

		// fake dns server
		opts = append(opts, ss.WithFakeDnsServer(cfg.Local.FakeDNS.Listen))
		opts = append(opts, ss.WithDefaultDnsNameservers(cfg.Local.FakeDNS.Nameservers))
	}
	if cfg.Local.SystemProxy {
		opts = append(opts, ss.WithSystemProxy())
	}

	return opts
}

func splitAuthInfo(auth string) (username, password string) {
	u, p, _ := strings.Cut(auth, ":")
	return u, p
}

func splitTunAddrInfo(addr string) ([]string, error) {
	local, remote, found := strings.Cut(addr, "=")
	if !found {
		return nil, fmt.Errorf("tun address invalid: %q", addr)
	}
	return []string{local, remote}, nil
}
