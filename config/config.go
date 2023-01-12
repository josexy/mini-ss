package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/ss"
)

type KcpOption struct {
	Crypt    string `json:"crypt"`
	Key      string `json:"key"`
	Mode     string `json:"mode"`
	Compress bool   `json:"compress"`
	Conns    int    `json:"conns"`
}

type WsOption struct {
	Host     string `json:"host"`
	Path     string `json:"path"`
	Compress bool   `json:"compress"`
	TLS      bool   `json:"tls"`
}

type ObfsOption struct {
	Host string `json:"host"`
	TLS  bool   `json:"tls"`
}

type QuicOption struct {
	Conns int `json:"conns"`
}

type SSROption struct {
	Protocol      string `json:"protocol"`
	ProtocolParam string `json:"protocol_param"`
	Obfs          string `json:"obfs"`
	ObfsParam     string `json:"obfs_param"`
}

type ServerJsonConfig struct {
	Disable   bool        `json:"disable"`
	Type      string      `json:"type,omitempty"`
	Name      string      `json:"name"`
	Addr      string      `json:"addr"`
	Password  string      `json:"password"`
	Method    string      `json:"method"`
	Transport string      `json:"transport"`
	Kcp       *KcpOption  `json:"kcp,omitempty"`
	Ws        *WsOption   `json:"ws,omitempty"`
	Obfs      *ObfsOption `json:"obfs,omitempty"`
	Quic      *QuicOption `json:"quic,omitempty"`
	SSR       *SSROption  `json:"ssr,omitempty"`
}

type TunOption struct {
	Name string `json:"name"`
	Cidr string `json:"cidr"`
	Mtu  int    `json:"mtu"`
}

type FakeDnsOption struct {
	Listen      string   `json:"listen"`
	Nameservers []string `json:"nameservers"`
}

type LocalJsonConfig struct {
	SocksAddr   string         `json:"socks_addr"`
	HTTPAddr    string         `json:"http_addr"`
	SocksAuth   string         `json:"socks_auth"`
	HTTPAuth    string         `json:"http_auth"`
	MixedAddr   string         `json:"mixed_addr"`
	TCPTunAddr  []string       `json:"tcp_tun_addr"`
	UDPTunAddr  []string       `json:"udp_tun_addr"`
	SystemProxy bool           `json:"system_proxy"`
	EnableTun   bool           `json:"enable_tun"`
	Tun         *TunOption     `json:"tun,omitempty"`
	FakeDNS     *FakeDnsOption `json:"fake_dns,omitempty"`
}

type Domain struct {
	Proxy  string   `json:"proxy"`
	Action string   `json:"action"`
	Value  []string `json:"value"`
}

type DomainKeyword struct {
	Proxy  string   `json:"proxy"`
	Action string   `json:"action"`
	Value  []string `json:"value"`
}

type DomainSuffix struct {
	Proxy  string   `json:"proxy"`
	Action string   `json:"action"`
	Value  []string `json:"value"`
}

type GeoIP struct {
	Resolve bool     `json:"resolve"`
	Proxy   string   `json:"proxy"`
	Action  string   `json:"action"`
	Value   []string `json:"value"`
}

type IPCidr struct {
	Resolve bool     `json:"resolve"`
	Proxy   string   `json:"proxy"`
	Action  string   `json:"action"`
	Value   []string `json:"value"`
}

type Match struct {
	Others         string           `json:"others,omitempty"`
	Domains        []*Domain        `json:"domain,omitempty"`
	DomainKeywords []*DomainKeyword `json:"domain_keyword,omitempty"`
	DomainSuffixs  []*DomainSuffix  `json:"domain_suffix,omitempty"`
	GeoIPs         []*GeoIP         `json:"geoip,omitempty"`
	IPCidrs        []*IPCidr        `json:"ipcidr,omitempty"`
}

type Rules struct {
	Mode     string `json:"mode"`
	DirectTo string `json:"direct_to"`
	GlobalTo string `json:"global_to"`
	Match    *Match `json:"match,omitempty"`
}

type JsonConfig struct {
	Server          []*ServerJsonConfig `json:"server,omitempty"`
	Local           *LocalJsonConfig    `json:"local,omitempty"`
	Color           bool                `json:"color"`
	Verbose         bool                `json:"verbose"`
	VerboseLevel    int                 `json:"verbose_level"`
	Iface           string              `json:"iface"`
	AutoDetectIface bool                `json:"auto_detect_iface"`
	Rules           *Rules              `json:"rules"`
}

func ParseJsonConfigFile(path string) (*JsonConfig, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	cfg := new(JsonConfig)
	if err = json.Unmarshal(data, cfg); err != nil {
		return nil, "", err
	}
	return cfg, string(data), nil
}

func (jsonCfg *JsonConfig) DeleteServerConfig(name string) {
	index := -1
	for i, c := range jsonCfg.Server {
		if c.Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}
	jsonCfg.Server = append(jsonCfg.Server[:index], jsonCfg.Server[index+1:]...)
}

func (jsonCfg *JsonConfig) BuildRuler() *dns.Ruler {
	var mode dns.RuleMode
	switch jsonCfg.Rules.Mode {
	case "global":
		mode = dns.Global
	case "direct":
		mode = dns.Direct
	default:
		mode = dns.Match
	}

	if mode == dns.Global || mode == dns.Direct {
		// glob and direct patterns do not require matching rules
		return dns.NewRuler(mode, "", "", nil)
	}

	// match mode
	var (
		domainRules        []dns.Rule
		domainKeywordRules []dns.Rule
		domainSuffixRules  []dns.Rule
		geoipRules         []dns.Rule
		ipcidrRules        []dns.Rule
		otherRules         []dns.Rule
	)
	for _, r := range jsonCfg.Rules.Match.Domains {
		domainRules = append(domainRules, dns.Rule{
			RuleMode: dns.Match,
			RuleType: dns.RuleDomain,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range jsonCfg.Rules.Match.DomainKeywords {
		domainKeywordRules = append(domainKeywordRules, dns.Rule{
			RuleMode: dns.Match,
			RuleType: dns.RuleDomainKeyword,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range jsonCfg.Rules.Match.DomainSuffixs {
		domainSuffixRules = append(domainSuffixRules, dns.Rule{
			RuleMode: dns.Match,
			RuleType: dns.RuleDomainSuffix,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range jsonCfg.Rules.Match.GeoIPs {
		geoipRules = append(geoipRules, dns.Rule{
			RuleMode: dns.Match,
			RuleType: dns.RuleGeoIP,
			Resolve:  r.Resolve,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}
	for _, r := range jsonCfg.Rules.Match.IPCidrs {
		ipcidrRules = append(ipcidrRules, dns.Rule{
			RuleMode: dns.Match,
			RuleType: dns.RuleIPCIDR,
			Resolve:  r.Resolve,
			Proxy:    r.Proxy,
			Accept:   r.Action == "accept",
			Value:    r.Value,
		})
	}

	var rules [][]dns.Rule
	otherRules = append(otherRules, dns.Rule{
		RuleMode: dns.Match,
		Proxy:    jsonCfg.Rules.Match.Others,
		RuleType: dns.RuleOthers,
		Accept:   true,
	})
	rules = append(rules, domainRules)
	rules = append(rules, domainKeywordRules)
	rules = append(rules, domainSuffixRules)
	rules = append(rules, geoipRules)
	rules = append(rules, ipcidrRules)
	rules = append(rules, otherRules)
	return dns.NewRuler(mode, jsonCfg.Rules.DirectTo, jsonCfg.Rules.GlobalTo, rules)
}

func (jsonCfg *JsonConfig) BuildSSLocalOptions() []ss.SSOption {
	opts := jsonCfg.BuildServerOptions()
	opts = append(opts, jsonCfg.BuildLocalOptions()...)
	return opts
}

func (jsonCfg *JsonConfig) BuildServerOptions() []ss.SSOption {
	var res []ss.SSOption

	for _, opt := range jsonCfg.Server {
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
			if opt.Obfs.TLS {
				opts = append(opts, ss.WithObfsTLS())
			}
		case "quic":
			opts = append(opts, ss.WithQuicTransport())
			opts = append(opts, ss.WithQuicConns(opt.Quic.Conns))
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

		res = append(res, ss.WithServerCompose(opts...))
	}

	// outbound interface
	res = append(res, ss.WithOutboundInterface(jsonCfg.Iface))
	// auto detect interface
	res = append(res, ss.WithAutoDetectInterface(jsonCfg.AutoDetectIface))
	return res
}

func (jsonCfg *JsonConfig) BuildLocalOptions() []ss.SSOption {
	var opts []ss.SSOption

	if jsonCfg.Local.SocksAddr != "" {
		opts = append(opts, ss.WithSocksAddr(jsonCfg.Local.SocksAddr))
		opts = append(opts, ss.WithSocksUserInfo(splitAuthInfo(jsonCfg.Local.SocksAuth)))
	}

	if jsonCfg.Local.HTTPAddr != "" {
		opts = append(opts, ss.WithHttpAddr(jsonCfg.Local.HTTPAddr))
		opts = append(opts, ss.WithHttpUserInfo(splitAuthInfo(jsonCfg.Local.HTTPAuth)))
	}

	opts = append(opts, ss.WithMixedAddr(jsonCfg.Local.MixedAddr))

	// simple tcp tun address
	var tcpTunAddr [][]string
	for _, addr := range jsonCfg.Local.TCPTunAddr {
		lr, _ := splitTunAddrInfo(addr)
		tcpTunAddr = append(tcpTunAddr, lr)
	}
	opts = append(opts, ss.WithTcpTunAddr(tcpTunAddr))

	// simple udp tun address
	var udpTunAddr [][]string
	for _, addr := range jsonCfg.Local.UDPTunAddr {
		lr, _ := splitTunAddrInfo(addr)
		udpTunAddr = append(udpTunAddr, lr)
	}
	opts = append(opts, ss.WithUdpTunAddr(udpTunAddr))
	opts = append(opts, ss.WithRuler(jsonCfg.BuildRuler()))

	if jsonCfg.Local.EnableTun {
		if jsonCfg.Local.FakeDNS == nil {
			logx.Fatal("if tun mode is enabled, the fake dns configuration must exist")
		}
		opts = append(opts, ss.WithEnableTun())
		opts = append(opts, ss.WithTunName(jsonCfg.Local.Tun.Name))
		opts = append(opts, ss.WithTunCIDR(jsonCfg.Local.Tun.Cidr))
		opts = append(opts, ss.WithTunMTU(uint32(jsonCfg.Local.Tun.Mtu)))

		// fake dns server
		opts = append(opts, ss.WithFakeDnsServer(jsonCfg.Local.FakeDNS.Listen))
		opts = append(opts, ss.WithDefaultDnsNameservers(jsonCfg.Local.FakeDNS.Nameservers))
	}
	if jsonCfg.Local.SystemProxy {
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
