package dns

import (
	"net"
	"net/netip"
	"strings"

	"github.com/fatih/color"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/socks/constant"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/ordmap"
)

type RuleType string

type RuleMode byte

const (
	Global RuleMode = iota
	Direct
	Match
)

var (
	RuleDomain        RuleType = "DOMAIN"
	RuleDomainKeyword RuleType = "DOMAIN-KEYWORD"
	RuleDomainSuffix  RuleType = "DOMAIN-SUFFIX"
	RuleGeoIP         RuleType = "GEOIP"
	RuleIPCIDR        RuleType = "IP-CIDR"
	RuleOthers        RuleType = "OTHERS"

	IndexToRuleType = []RuleType{
		RuleDomain,
		RuleDomainKeyword,
		RuleDomainSuffix,
		RuleGeoIP,
		RuleIPCIDR,
		RuleOthers,
	}
	RuleTypeToIndex = map[RuleType]int{
		RuleDomain:        0,
		RuleDomainKeyword: 1,
		RuleDomainSuffix:  2,
		RuleGeoIP:         3,
		RuleIPCIDR:        4,
		RuleOthers:        5,
	}
)

func (m RuleMode) String() string {
	switch m {
	case Global:
		return "GLOBAL"
	case Direct:
		return "DIRECT"
	default:
		return "MATCH"
	}
}

type Ruler struct {
	RuleMode
	mr       Rule
	M        []Matcher
	DirectTo string // direct strategy for MATCH mode
	GlobalTo string // global strategy for MATCH mode
}

func NewRuler(mode RuleMode, directTo, globalTo string, rules [][]Rule) *Ruler {
	var ms []Matcher
	for i, rs := range rules {
		ms = append(ms, NewRuleMatcher(IndexToRuleType[i], rs))
		logx.Info("register [%s] match-ruler", IndexToRuleType[i])
	}
	if directTo == "direct" || directTo == "global" {
		directTo = ""
	}
	if globalTo == "global" || globalTo == "direct" {
		globalTo = ""
	}
	return &Ruler{
		RuleMode: mode,
		M:        ms,
		DirectTo: directTo,
		GlobalTo: globalTo,
	}
}

// Match global/direct/match
// the target value may be:
// 1. real ip address -> match
// 2. fake ip address -> domain name -> match
// 3. domain name -> match
func (r *Ruler) Match(target *string) (drop bool) {
	r.mr = Rule{}
	if r.RuleMode == Global || r.RuleMode == Direct {
		r.mr = Rule{
			RuleMode: r.RuleMode,
			Proxy:    "auto-select",
			Accept:   true,
		}
		return false
	}
	for _, matcher := range r.M {
		if matcher.Match(target) {
			// returning true means discarding the request, otherwise accepting the request
			r.mr = matcher.MatchedResult()
			drop = !r.mr.Accept
			if drop {
				logx.Warn("[%s] %s/%s/%s for %s", color.RedString("ruler-drop"),
					color.GreenString(r.mr.RuleMode.String()),
					color.YellowString("%s", r.mr.RuleType),
					color.RedString(r.mr.Proxy),
					color.BlueString(*target))
			}
			return
		}
	}
	// other miss rules
	r.mr = Rule{
		RuleMode: r.RuleMode,
		RuleType: RuleOthers,
	}
	return false
}

func (r *Ruler) SelectOne(sels *ordmap.OrderedMap, conn net.PacketConn, dstAddr string) error {
	var err error

	if r.mr.Accept {
		logx.Warn("[%s] %s/%s/%s for %s", color.GreenString("ruler-accept"),
			color.GreenString(r.RuleMode.String()),
			color.YellowString("%s", r.mr.RuleType),
			color.RedString(r.mr.Proxy),
			color.BlueString(dstAddr))
	}

	globalFn := func() {
		if value, ok := sels.First(); ok {
			relayer := value.(transport.DstAddrRelayer)
			err = relayer.RelayUDP(conn, relayer.DstAddr, dstAddr)
		}
	}

	directFn := func() {
		if value, ok := sels.First(); ok {
			relayer := value.(transport.DstAddrRelayer)
			err = relayer.RelayDirectUDP(conn, dstAddr)
		}
	}

	switch r.RuleMode {
	case Global:
		globalFn()
	case Direct:
		directFn()
	case Match:
		switch {
		case r.mr.Proxy == "global":
			// select a proxy and relay udp packet to it
			globalFn()
		default:
			// connect directly to the udp server
			directFn()
		}
	}
	return err
}

func (r *Ruler) Select(sels *ordmap.OrderedMap, conn net.Conn, dstAddr string) error {
	defer func() { recover() }()

	var err error
	if r.mr.Accept {
		logx.Warn("[%s] %s/%s/%s for %s", color.GreenString("ruler-accept"),
			color.GreenString(r.mr.RuleMode.String()),
			color.YellowString("%s", r.mr.RuleType),
			color.RedString(r.mr.Proxy),
			color.BlueString(dstAddr))
	} else {
		logx.Warn("[%s] %s/%s/%s for %s", color.RedString("ruler-others-drop"),
			color.GreenString(r.mr.RuleMode.String()),
			color.YellowString("%s", r.mr.RuleType),
			color.RedString(r.mr.Proxy),
			color.BlueString(dstAddr))
	}

	globalFn := func() {
		sels.Range(func(key, value any) bool {
			relayer := value.(transport.DstAddrRelayer)
			logx.Info("Using Proxy: %q:%q", key.(string), relayer.DstAddr)
			if err = relayer.RelayTCP(conn, relayer.DstAddr, dstAddr); err == nil {
				return false
			} else {
				logx.Error("[GLOBAL] error occurred at (%v:%s->%s), find next proxy, err: %v", key, relayer.DstAddr, dstAddr, err)
				return true
			}
		})
	}

	directFn := func() {
		// for DIRECT mode, dstAddr may be domain name or ip address
		// if dstAddr is domain name, it needs to be resolved to a real ip address
		host, port, _ := net.SplitHostPort(dstAddr)
		if net.ParseIP(host) == nil {
			// the dstAddr is domain name
			var ip netip.Addr
			switch resolver.DefaultResolver.IsFakeIPMode() {
			case true:
				// resolve domain name to real ip address
				if record := resolver.DefaultResolver.Find(host); record != nil {
					ip = resolver.DefaultResolver.ResolveQuery(record.Query)
					record.RealIP = ip
					break
				}
				fallthrough
			default:
				// resolve domain name to real ip address
				ip = resolver.DefaultResolver.ResolveHost(host)
			}

			if !ip.IsValid() {
				return
			}
			dstAddr = net.JoinHostPort(ip.String(), port)
		}

		if value, ok := sels.First(); ok {
			relayer := value.(transport.DstAddrRelayer)
			if err = relayer.RelayDirectTCP(conn, dstAddr); err != nil {
				logx.Error("[DIRECT] error occurred at (%s), err: %v", dstAddr, err)
			}
		}
	}

	selectFn := func(proxy string) (err error) {
		if v, ok := sels.Load(proxy); ok {
			relayer := v.(transport.DstAddrRelayer)
			err = relayer.RelayTCP(conn, relayer.DstAddr, dstAddr)
		} else {
			err = constant.ErrRuleMatchDropped
		}
		return
	}

	// update connection statistic status information
	addr := conn.RemoteAddr().String()
	statistic.DefaultManager.LazySet(addr, statistic.LazyContext{
		Host:     dstAddr,
		RuleMode: r.mr.RuleMode.String(),
		RuleType: string(r.mr.RuleType),
		Proxy:    r.mr.Proxy,
	})

	switch r.mr.RuleMode {
	case Global:
		globalFn()
	case Direct:
		directFn()
	case Match:
		switch {
		case r.mr.Proxy == "global":
			if r.GlobalTo != "" {
				err = selectFn(r.GlobalTo)
			} else {
				globalFn()
			}
		case r.mr.Proxy == "direct":
			if r.DirectTo != "" {
				err = selectFn(r.DirectTo)
			} else {
				directFn()
			}
		case r.mr.Proxy != "":
			err = selectFn(r.mr.Proxy)
		default:
			err = constant.ErrRuleMatchDropped
		}
	}
	return err
}

func (r *Ruler) MatcherResult() Rule {
	return r.mr
}

type Matcher interface {
	Match(*string) bool
	MatchedResult() Rule
}

type Rule struct {
	RuleMode
	RuleType
	Resolve bool
	Proxy   string
	Accept  bool
	Value   []string
}

type domainRule struct {
	mr Rule
	R  []Rule
}

type domainSuffixRule struct {
	mr Rule
	R  []Rule
}

type domainKeywordRule struct {
	mr Rule
	R  []Rule
}

type geoipRule struct {
	mr Rule
	R  []Rule
}

type ipCIDRRule struct {
	mr Rule
	R  []Rule
}

type otherRule struct {
	R Rule
}

func NewRuleMatcher(ruleType RuleType, rules []Rule) Matcher {
	switch ruleType {
	case RuleDomain:
		return &domainRule{R: rules}
	case RuleDomainSuffix:
		return &domainSuffixRule{R: rules}
	case RuleDomainKeyword:
		return &domainKeywordRule{R: rules}
	case RuleGeoIP:
		return &geoipRule{R: rules}
	case RuleIPCIDR:
		return &ipCIDRRule{R: rules}
	case RuleOthers:
		return &otherRule{R: rules[0]}
	}
	return nil
}

func (r *domainRule) Match(target *string) bool {
	r.mr = Rule{}
	for _, rx := range r.R {
		for _, rule := range rx.Value {
			if *target == rule {
				r.mr = rx
				r.mr.Value = r.mr.Value[:0]
				return true
			}
		}
	}
	return false
}

func (r *domainRule) MatchedResult() Rule {
	return r.mr
}

func (r *domainSuffixRule) Match(target *string) bool {
	r.mr = Rule{}
	for _, rx := range r.R {
		for _, rule := range rx.Value {
			if strings.HasSuffix(*target, rule) {
				r.mr = rx
				r.mr.Value = r.mr.Value[:0]
				return true
			}
		}
	}
	return false
}

func (r *domainSuffixRule) MatchedResult() Rule {
	return r.mr
}

func (r *domainKeywordRule) Match(target *string) bool {
	r.mr = Rule{}
	for _, rx := range r.R {
		for _, rule := range rx.Value {
			if strings.Contains(*target, rule) {
				r.mr = rx
				r.mr.Value = r.mr.Value[:0]
				return true
			}
		}
	}
	return false
}

func (r *domainKeywordRule) MatchedResult() Rule {
	return r.mr
}

func (r *geoipRule) Match(target *string) bool {
	r.mr = Rule{}
	for _, rx := range r.R {
		// resolve domain name to ip address
		if rx.Resolve {
			if _, err := netip.ParseAddr(*target); err != nil {
				ip := resolver.DefaultResolver.ResolveHost(*target)
				if ip.IsValid() {
					*target = ip.String()
				}
			}
		}
		ip, _ := netip.ParseAddr(*target)
		for _, rule := range rx.Value {
			if rule == geoip.QueryCountryByIP(ip) {
				r.mr = rx
				r.mr.Value = r.mr.Value[:0]
				return true
			}
		}
	}
	return false
}

func (r *geoipRule) MatchedResult() Rule {
	return r.mr
}

func (r *ipCIDRRule) Match(target *string) bool {
	r.mr = Rule{}
	for _, rx := range r.R {
		// resolve domain name to ip address
		if rx.Resolve {
			if _, err := netip.ParseAddr(*target); err != nil {
				ip := resolver.DefaultResolver.ResolveHost(*target)
				if ip.IsValid() {
					*target = ip.String()
				}
			}
		}
		ip, _ := netip.ParseAddr(*target)
		for _, rule := range rx.Value {
			subnet := netip.MustParsePrefix(rule)
			if subnet.Contains(ip) {
				r.mr = rx
				r.mr.Value = r.mr.Value[:0]
				return true
			}
		}
	}
	return false
}

func (r *ipCIDRRule) MatchedResult() Rule {
	return r.mr
}

func (r *otherRule) Match(*string) bool {
	return r.R.Proxy != ""
}

func (r *otherRule) MatchedResult() Rule {
	return r.R
}
