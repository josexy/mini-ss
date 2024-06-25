package resolver

import (
	"errors"
	"net/netip"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/cache"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/miekg/dns"
)

var (
	DefaultFakeIPDnsRecordTTL  = 60 * time.Second
	DefaultFakeIPCacheInterval = 30 * time.Second
)

type Record struct {
	Domain string
	FakeIP netip.Addr
	Query  *dns.Msg
	Reply  *dns.Msg
}

type fakeIPResolver struct {
	// fake ip pool
	pool *ipPool
	// host:record
	cache cache.Cache[string, *Record]
	// ip:host
	ipCache   cache.Cache[netip.Addr, *Record]
	dnsIP     netip.Addr
	tunPrefix netip.Prefix
}

func newFakeIPResolver(cidr netip.Prefix) (*fakeIPResolver, error) {
	pool, err := newIPPool(cidr)
	if err != nil {
		return nil, err
	}

	// pre-allocated tun ip and fake dns ip
	tunIP, ok := pool.allocateFor(cidr.Addr())
	if !ok {
		return nil, errors.New("can not allocate ip for tun device")
	}
	dnsIP, ok := pool.allocateFor(tunIP.Next())
	if !ok {
		logger.Logger.ErrorBy(errors.New("can not allocate ip for fake dns"))
	}
	logger.Logger.Infof("pre-allocated ip for tun device: %s, dns: %s", tunIP.String(), dnsIP.String())

	r := &fakeIPResolver{
		pool:      pool,
		dnsIP:     dnsIP,
		tunPrefix: netip.PrefixFrom(tunIP, pool.Bits()),
	}
	r.cache = cache.NewCache[string, *Record](
		cache.WithMaxSize(4096),
		cache.WithInterval(DefaultFakeIPCacheInterval),
		cache.WithExpiration(DefaultFakeIPDnsRecordTTL),
		cache.WithEvictCallback(r.onReleaseFakeIP),
		cache.WithDeleteExpiredCacheOnGet(),
		cache.WithBackgroundCheckCache(),
	)
	r.ipCache = cache.NewCache[netip.Addr, *Record](
		cache.WithMaxSize(4096),
		cache.WithInterval(DefaultFakeIPCacheInterval),
		cache.WithExpiration(DefaultFakeIPDnsRecordTTL),
		cache.WithDeleteExpiredCacheOnGet(),
		cache.WithBackgroundCheckCache(),
	)
	return r, nil
}

func (r *fakeIPResolver) onReleaseFakeIP(_ any, value any) {
	record := value.(*Record)
	logger.Logger.Trace("release fake ip", logx.String("ip", record.FakeIP.String()), logx.String("domain", record.Domain))
	r.pool.Release(record.FakeIP)
}

func (r *fakeIPResolver) GetAllocatedTunPrefix() netip.Prefix { return r.tunPrefix }

func (r *fakeIPResolver) GetAllocatedDnsIP() netip.Addr { return r.dnsIP }

func (r *fakeIPResolver) IsFakeIP(ip netip.Addr) bool { return r.pool.Contains(ip) }

func (r *fakeIPResolver) find(host string) (*Record, error) { return r.cache.Get(host) }

func (r *fakeIPResolver) FindByIP(ip netip.Addr) (*Record, error) {
	if host, err := r.ipCache.Get(ip); err == nil {
		return r.find(host.Domain)
	} else {
		return nil, err
	}
}

func (r *fakeIPResolver) makeNewFakeDnsRecord(host string, request *dns.Msg) (*Record, error) {
	// allocate a fake ip for dns query host
	fakeIP, err := r.pool.Allocate(host)
	if err != nil {
		return nil, err
	}
	if !fakeIP.IsValid() {
		return nil, errors.New("unable to allocate fake ip from pool")
	}
	reply := &dns.Msg{}
	reply.SetReply(request)
	reply.RecursionAvailable = true
	reply.Answer = append(reply.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(host),
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    uint32(DefaultFakeIPDnsRecordTTL.Seconds()),
		},
		A: fakeIP.AsSlice(),
	})

	record := &Record{
		Domain: host,
		FakeIP: fakeIP,
		Query:  request,
		Reply:  reply,
	}
	// save to cache
	r.cache.Set(host, record)
	r.ipCache.Set(fakeIP, record)
	logger.Logger.Trace("allocate fake ip", logx.String("ip", fakeIP.String()), logx.String("domain", host))
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
	// dns record exists in cache
	if record, err := r.find(domain); err == nil {
		record.Reply.SetReply(req.Copy())
		return record.Reply, nil
	}
	// dns record dose not exist and create a new record
	record, err := r.makeNewFakeDnsRecord(domain, req.Copy())
	if err != nil {
		return nil, err
	}
	return record.Reply, nil
}
