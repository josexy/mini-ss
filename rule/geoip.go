package rule

import (
	"context"
	"net/netip"

	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/resolver"
)

type geoipRule struct {
	result *RuleItem
	R      []*RuleItem
}

func (r *geoipRule) Match(target *string) bool {
	for _, rx := range r.R {
		// if it is a domain name, resolve it to get the IP address, and then match it
		if rx.Resolve {
			if _, err := netip.ParseAddr(*target); err != nil {
				ip := resolver.DefaultResolver.LookupHost(context.Background(), *target)
				if ip.IsValid() {
					*target = ip.String()
				}
			}
		}

		ip, _ := netip.ParseAddr(*target)
		for _, rule := range rx.Value {
			if rule == geoip.QueryCountryByIP(ip) {
				r.result = rx
				return true
			}
		}
	}
	return false
}

func (r *geoipRule) MatchedResult() *RuleItem {
	return r.result
}
