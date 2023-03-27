package rule

import "strings"

type domainKeywordRule struct {
	result *RuleItem
	R      []*RuleItem
}

func (r *domainKeywordRule) Match(target *string) bool {
	if target == nil || len(*target) == 0 {
		return false
	}
	for _, rx := range r.R {
		for _, rule := range rx.Value {
			if strings.Contains(*target, rule) {
				r.result = rx
				return true
			}
		}
	}
	return false
}

func (r *domainKeywordRule) MatchedResult() *RuleItem {
	return r.result
}
