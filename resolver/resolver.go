package resolver

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/hostsutil"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/miekg/dns"
)

var (
	errCannotLookupIPFromHostsFile = errors.New("cannot lookup ip from local hosts file")
	errCannotLookupIPv4v6          = errors.New("cannot lookup ipv4 and ipv6")
	errDnsExchangedFailed          = errors.New("dns exchanged failed")
)

var DefaultResolver *Resolver

type Resolver struct {
	*fakeIPResolver
	client      *dns.Client
	nameservers []string
}

func NewDnsResolver(nameservers []string) *Resolver {
	// read local dns configuration
	localDnsList := dnsutil.GetLocalDnsList()
	nameservers = append(nameservers, localDnsList...)
	var nss []string
	for i := 0; i < len(nameservers); i++ {
		host, port, _ := net.SplitHostPort(nameservers[i])
		if host == "" || port == "" {
			host = nameservers[i]
			port = "53"
		}
		if ip, err := netip.ParseAddr(host); err == nil && (ip.Is4() || ip.Is6() || ip.Is4In6()) {
			addr := net.JoinHostPort(host, port)
			nss = append(nss, addr)
			logger.Logger.Infof("dns nameserver: %s", addr)
		}
	}

	return &Resolver{
		nameservers: nss,
		client: &dns.Client{
			Net:          "", // UDP
			UDPSize:      4096,
			Timeout:      2 * time.Second,
			ReadTimeout:  2 * time.Second,
			WriteTimeout: 2 * time.Second,
		},
	}
}

func (r *Resolver) IsEnhancerMode() bool {
	return r.fakeIPResolver != nil
}

func (r *Resolver) EnableEnhancerMode(tunCIDR string) (err error) {
	r.fakeIPResolver, err = newFakeIPResolver(tunCIDR)
	return
}

func (r *Resolver) LookupHost(ctx context.Context, host string) netip.Addr {
	if host == "" {
		return netip.Addr{}
	}
	// check if the host is ip address
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip
	}
	addrs, err := r.LookupIP(ctx, host)
	if err != nil || len(addrs) == 0 {
		return netip.Addr{}
	}
	return addrs[rand.Intn(len(addrs))]
}

func (r *Resolver) LookupIP(ctx context.Context, host string) ([]netip.Addr, error) {
	ipsCh := make(chan []netip.Addr, 1)
	// lookup ipv6 address
	go func() {
		defer close(ipsCh)
		ips, err := r.lookupIP(ctx, host, dns.TypeAAAA)
		if err != nil {
			return
		}
		ipsCh <- ips
	}()
	// lookup ipv4 address
	ips, err := r.lookupIP(ctx, host, dns.TypeA)
	if err == nil {
		return ips, nil
	}
	ips, ok := <-ipsCh
	if !ok {
		return nil, errCannotLookupIPv4v6
	}
	return ips, nil
}

func (r *Resolver) lookupIP(ctx context.Context, host string, dnsType uint16) ([]netip.Addr, error) {
	req := &dns.Msg{}
	req.SetQuestion(dns.Fqdn(host), dnsType)
	req.RecursionDesired = true
	reply, err := r.exchangeContext(ctx, req)
	if err != nil {
		return nil, err
	}
	return dnsutil.MsgToAddrs(reply), nil
}

func (r *Resolver) exchangeContext(ctx context.Context, req *dns.Msg) (msg *dns.Msg, err error) {
	msg, err = r.exchangeContextWithoutCache(ctx, req)
	return
}

func (r *Resolver) exchangeContextWithoutCache(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	// request the dns server one after another.
	// once a dns returns a reply, it returns immediately.
	replyCh := make(chan *dns.Msg, 1)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	wg := sync.WaitGroup{}

	getReplyDnsMsg := func(ctx context.Context, nameserver string) {
		defer wg.Done()
		logger.Logger.Tracef("dns query via %s", nameserver)
		reply, _, err := r.client.ExchangeContext(ctx, req, nameserver)
		if err != nil {
			return
		}
		if reply.Rcode != dns.RcodeSuccess {
			return
		}
		select {
		case replyCh <- reply:
			logger.Logger.Tracef("dns query via %s succeed", nameserver)
		default:
		}
	}

	for _, nameserver := range r.nameservers {
		wg.Add(1)
		go getReplyDnsMsg(ctx, nameserver)
		select {
		case reply := <-replyCh:
			return reply, nil
		case <-ticker.C:
			// if timeout, try the next dns nameserver
			continue
		}
	}

	wg.Wait()
	select {
	case reply := <-replyCh:
		return reply, nil
	default:
		return nil, errDnsExchangedFailed
	}
}

func (r *Resolver) lookupHostsFile(req *dns.Msg) (*dns.Msg, error) {
	host := dnsutil.TrimDomain(req.Question[0].Name)
	ip := hostsutil.LookupIP(host)
	if !ip.IsValid() {
		return nil, errCannotLookupIPFromHostsFile
	}
	reply := new(dns.Msg)
	reply.SetReply(req)
	reply.RecursionAvailable = true

	var rr dns.RR
	if ip.Is4() {
		rr = &dns.A{
			Hdr: dns.RR_Header{
				Name:   dns.Fqdn(host),
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    6,
			},
			A: ip.AsSlice(),
		}
	} else {
		rr = &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   dns.Fqdn(host),
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    6,
			},
			AAAA: ip.AsSlice(),
		}
	}
	reply.Answer = append(reply.Answer, rr)
	return reply, nil
}

func (r *Resolver) Query(req *dns.Msg) (reply *dns.Msg, err error) {
	reply, err = r.lookupHostsFile(req)
	if err == nil {
		return
	}

	if req.Question[0].Qclass == dns.ClassINET && req.Question[0].Qtype == dns.TypeA {
		// ipv4 dns query, return fake ip address
		reply, err = r.fakeIPResolver.query(req)
	} else {
		// return an empty response for ipv6
		reply, err = &dns.Msg{}, nil
		reply.SetReply(req)
	}
	return
}
