package rule

import (
	"errors"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/logger"
)

const (
	Global RuleMode = iota
	Direct
	Match
)

type (
	RuleType string
	RuleMode uint8

	RuleItem struct {
		RuleMode
		RuleType
		Proxy   string
		Value   []string
		Resolve bool
		Accept  bool
	}
)

var (
	errEmptyGlobalProxyNode = errors.New("empty global proxy node")
	ErrRuleMatchDropped     = errors.New("rule matched dropped")
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

var MatchRuler *Ruler

type Ruler struct {
	RuleMode
	matched  *RuleItem
	MS       []Matcher
	DirectTo string // direct connection strategy for MATCH mode
	GlobalTo string // global connection strategy for MATCH mode
}

func NewRuler(mode RuleMode, directTo, globalTo string, allRules [][]*RuleItem) *Ruler {
	var matchers []Matcher
	for i, rules := range allRules {
		matchers = append(matchers, newRuleMatcher(IndexToRuleType[i], rules))
		logger.Logger.Infof("register [%s] match-ruler", IndexToRuleType[i])
	}
	if directTo == "direct" || directTo == "global" {
		directTo = ""
	}
	if globalTo == "global" || globalTo == "direct" {
		globalTo = ""
	}
	return &Ruler{
		MS:       matchers,
		RuleMode: mode,
		DirectTo: directTo,
		GlobalTo: globalTo,
	}
}

// Match global/direct/match
// the target value may be:
// 1. real ip address -> match
// 2. fake ip address -> domain name -> match
// 3. domain name -> match
func (r *Ruler) Match(target *string) bool {
	if r.RuleMode == Global || r.RuleMode == Direct {
		r.matched = &RuleItem{
			RuleMode: r.RuleMode,
			Proxy:    "auto-select",
			Accept:   true,
		}
		return true
	}
	for _, matcher := range r.MS {
		if matcher.Match(target) {
			// return true to discard the request, otherwise accept
			r.matched = matcher.MatchedResult()
			if !r.matched.Accept {
				logger.Logger.Error("request dropped",
					logx.String("mode", r.matched.RuleMode.String()),
					logx.String("type", string(r.matched.RuleType)),
					logx.String("target", *target),
				)
				return false
			}
			r.matched = matcher.MatchedResult()
			logger.Logger.Info("match success",
				logx.String("mode", r.matched.RuleMode.String()),
				logx.String("type", string(r.matched.RuleType)),
				logx.String("proxy", r.matched.Proxy),
				logx.String("target", *target),
			)
			return true
		}
	}
	// oops!
	return false
}

func (r *Ruler) MatcherResult() *RuleItem { return r.matched }

// Select select a valid proxy node from the currently matched rules
// if err is not equal to nil, discard the request
// the returned proxy indicates whether the selected rule is global or direct connection
// if the proxy is empty, it means the proxy is directly connected
func (r *Ruler) Select() (proxy string, err error) {
	defer func() {
		logger.Logger.Info("proxy selected", logx.String("proxy", proxy))
	}()
	mode := r.matched.RuleMode
	if mode == Match {
		switch {
		case r.matched.Proxy == "global":
			mode = Global
			goto _global
		case r.matched.Proxy == "direct":
			mode = Direct
			goto _direct
		case r.matched.Proxy != "":
			return r.matched.Proxy, nil
		default:
			return "", ErrRuleMatchDropped
		}
	}

_global:
	if mode == Global {
		proxy = r.GlobalTo
		// a proxy node should be provided for global mode
		if proxy == "" {
			err = errEmptyGlobalProxyNode
		}
		return
	}

_direct:
	if mode == Direct {
		proxy = r.DirectTo
	}
	return
}
