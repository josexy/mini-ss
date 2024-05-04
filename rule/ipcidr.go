package rule

import (
	"context"
	"net/netip"

	"github.com/josexy/mini-ss/resolver"
)

type ipCIDRRule struct {
	result *RuleItem
	R      []*RuleItem
}

func (r *ipCIDRRule) Match(target *string) bool {
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
			subnet := netip.MustParsePrefix(rule)
			if subnet.Contains(ip) {
				r.result = rx
				return true
			}
		}
	}
	return false
}

func (r *ipCIDRRule) MatchedResult() *RuleItem {
	return r.result
}
