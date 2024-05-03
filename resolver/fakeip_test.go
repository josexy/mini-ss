package resolver

import (
	"net/netip"
	"testing"
	"time"

	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func doQuery(t *testing.T, r *fakeIPResolver, host string) netip.Addr {
	req := new(dns.Msg)
	req.SetQuestion(dns.Fqdn(host), dns.TypeA)
	req.RecursionDesired = true

	fakeReply, err := r.query(req)
	assert.Nil(t, err)
	t.Logf("dns reply host: %s, ttl: %d", host, fakeReply.Answer[0].Header().Ttl)
	return dnsutil.MsgToIP(fakeReply)[0]
}

func TestFakeIPResolverQueryTTL(t *testing.T) {
	DefaultFakeIPCacheInterval = time.Millisecond * 1500
	DefaultFakeIPDnsRecordTTL = time.Second
	r, err := newFakeIPResolver("198.18.2.22/24")
	assert.Nil(t, err)

	host := "www.example.com"
	// cache not found and create a new one
	oldFakeIP := doQuery(t, r, host)
	t.Log(oldFakeIP)
	time.Sleep(time.Millisecond * 800)

	// hit cache
	hitOldFakeIP := doQuery(t, r, host)
	t.Log(hitOldFakeIP)
	assert.Equal(t, oldFakeIP, hitOldFakeIP)
	time.Sleep(time.Millisecond * 400)

	// ttl expired and fake ip released, cache not found and create a new one
	newFakeIP := doQuery(t, r, host)
	t.Log(newFakeIP)
	assert.Equal(t, oldFakeIP, newFakeIP)
}
