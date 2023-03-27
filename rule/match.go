package rule

type Matcher interface {
	Match(*string) bool
	MatchedResult() *RuleItem
}

func newRuleMatcher(ruleType RuleType, rules []*RuleItem) Matcher {
	switch ruleType {
	case RuleDomain:
		return newDomainRule(rules)
	case RuleDomainSuffix:
		return newDomainSuffixRule(rules)
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
