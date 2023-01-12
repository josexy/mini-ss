package resolver

import (
	"errors"
	"net/netip"

	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/miekg/dns"
)

const fakeIPDnsRecordTTL uint32 = 60

type Record struct {
	Domain string
	RealIP netip.Addr
	FakeIP netip.Addr
	Query  *dns.Msg
	Reply  *dns.Msg
}

type fakeIPResolver struct {
	// fake ip pool
	pool *fakeIPPool
	// host:record
	cache *LruCache
	// ip:host
	ipCache *LruCache
}

func newFakeIPResolver(cidr string) *fakeIPResolver {
	r := &fakeIPResolver{
		pool: newFakeIPPool(cidr),
	}
	r.cache = newLruCache(
		WithSize(4096),
		WithStale(false),
		WithUpdateAgeOnGet(),
		WithAge(int64(fakeIPDnsRecordTTL)),
		WithEvict(EvictCallback(r.onReleaseFakeIP)),
	)
	r.ipCache = newLruCache(
		WithSize(4096),
		WithStale(false),
		WithUpdateAgeOnGet(),
		WithAge(int64(fakeIPDnsRecordTTL)),
	)
	return r
}

func (r *fakeIPResolver) IsFakeIPMode() bool {
	return r.pool != nil
}

func (r *fakeIPResolver) onReleaseFakeIP(_ any, value any) {
	r.pool.Release(value.(*Record).FakeIP)
}

func (r *fakeIPResolver) Find(host string) *Record {
	if value, ok := r.cache.Get(host); ok {
		return value.(*Record)
	}
	return nil
}

func (r *fakeIPResolver) FindByIP(ip netip.Addr) *Record {
	if value, ok := r.ipCache.Get(ip); ok {
		return r.Find(value.(string))
	}
	return nil
}

func (r *fakeIPResolver) makeFakeDnsRecord(host string, req *dns.Msg) (*Record, error) {
	ip := r.pool.Alloc(host)
	if !ip.IsValid() {
		return nil, errors.New("unable to allocate fake ip from pool")
	}
	reply := &dns.Msg{}
	reply.SetReply(req)
	reply.RecursionAvailable = true
	reply.Answer = append(reply.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(host),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    fakeIPDnsRecordTTL,
		},
		A: ip.AsSlice(),
	})

	record := &Record{
		Domain: host,
		FakeIP: ip,
		Query:  req,
		Reply:  reply,
	}
	// save to lru cache
	r.cache.Set(host, record)
	r.ipCache.Set(ip, host)
	return record, nil
}

// query The local DNS request returns a FakeIP address,
// and the returned IP address is limited to the network segment of the TUN device,
// so that the traffic sent by the application is sent to the TUN device network,
// and at the same time, the routes that do not need to be matched can be excluded
// to ensure that some traffic can be sent directly to remote IP.
// For example, the traffic initiated by the proxy client ss-local
// needs routing exclusion to ensure that it can be sent directly to the remote server ss-server
// instead of directly to the TUN device, otherwise it will cause a loop
func (r *fakeIPResolver) query(req *dns.Msg) (*dns.Msg, error) {
	domain := dnsutil.TrimDomain(req.Question[0].Name)

	value, hit := r.cache.Get(domain)
	if hit {
		record := value.(*Record)
		record.Reply.SetReply(req)
		return record.Reply, nil
	}
	// dns record dose not exist
	record, err := r.makeFakeDnsRecord(domain, req.Copy())
	if err != nil {
		return nil, err
	}
	return record.Reply, nil
}
