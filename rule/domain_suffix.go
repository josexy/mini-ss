package rule

import (
	"github.com/josexy/mini-ss/util/trie"
)

type domainSuffixRule struct {
	result *RuleItem
	t      *trie.DomainTrie
}

func newDomainSuffixRule(rules []*RuleItem) *domainSuffixRule {
	r := &domainSuffixRule{t: trie.New()}
	for i := 0; i < len(rules); i++ {
		for j := 0; j < len(rules[i].Value); j++ {
			r.t.Insert("+."+rules[i].Value[j], rules[i])
		}
	}
	return r
}

func (r *domainSuffixRule) Match(target *string) bool {
	if target == nil || len(*target) == 0 {
		return false
	}
	res := r.t.Search(*target)
	if res == nil {
		return false
	}
	r.result = res.Data.(*RuleItem)
	return true
}

func (r *domainSuffixRule) MatchedResult() *RuleItem {
	return r.result
}
