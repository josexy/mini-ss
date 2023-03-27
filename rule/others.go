package rule

type otherRule struct {
	R *RuleItem
}

func (r *otherRule) Match(*string) bool {
	return r.R.Proxy != ""
}

func (r *otherRule) MatchedResult() *RuleItem {
	return r.R
}
