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

var DefaultResolver *Resolver

type Resolver struct {
	*fakeIPResolver
	client      *dns.Client
	nameservers []string
}

func NewDnsResolver(nameservers []string) *Resolver {
	// read local dns configuration
	localDnsList := dnsutil.GetLocalDnsList()
	nameservers = append(localDnsList, nameservers...)
	var ns []string
	for i := 0; i < len(nameservers); i++ {
		host, _, _ := net.SplitHostPort(nameservers[i])
		if ip, err := netip.ParseAddr(host); err == nil && ip.Is4() {
			ns = append(ns, nameservers[i])
		}
	}
	for i := 0; i < len(ns); i++ {
		host, port, _ := net.SplitHostPort(ns[i])
		if port == "" {
			ns[i] = net.JoinHostPort(host, "53")
		}
		logger.Logger.Infof("dns nameserver: %s", ns[i])
	}

	return &Resolver{
		nameservers: ns,
		client: &dns.Client{
			Net:          "", // UDP
			UDPSize:      4096,
			Timeout:      5 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		},
	}
}

func (r *Resolver) IsEnhancerMode() bool {
	return r.fakeIPResolver != nil
}

func (r *Resolver) EnableEnhancerMode(tunCIDR string) {
	r.fakeIPResolver = newFakeIPResolver(tunCIDR)
}

func (r *Resolver) ResolveQuery(req *dns.Msg) netip.Addr {
	reply, err := r.exchangeContext(context.Background(), req)
	if err != nil {
		return netip.Addr{}
	}
	ips := dnsutil.MsgToIP(reply)
	if len(ips) == 0 {
		return netip.Addr{}
	}
	return ips[rand.Intn(len(ips))]
}

func (r *Resolver) ResolveHost(host string) netip.Addr {
	// the host is ip address
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip
	}
	ips, err := r.LookupIP(context.Background(), host)
	if err != nil || len(ips) == 0 {
		return netip.Addr{}
	}
	return ips[rand.Intn(len(ips))]
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
		// can not lookup ipv4 and ipv6 address
		return nil, errors.New("can not lookup host")
	}
	return ips, nil
}

// lookupIP resolve host to IPv4 or IPv6 address
func (r *Resolver) lookupIP(ctx context.Context, host string, dnsType uint16) ([]netip.Addr, error) {
	req := &dns.Msg{}
	req.SetQuestion(dns.Fqdn(host), dnsType)
	req.RecursionDesired = true
	reply, err := r.exchangeContext(ctx, req)
	if err != nil {
		return nil, err
	}
	return dnsutil.MsgToIP(reply), nil
}

func (r *Resolver) exchange(req *dns.Msg) (*dns.Msg, error) {
	return r.exchangeContext(context.Background(), req)
}

func (r *Resolver) exchangeContext(ctx context.Context, req *dns.Msg) (msg *dns.Msg, err error) {
	msg, err = r.exchangeContextWithoutCache(ctx, req)
	return
}

// exchangeContextWithoutCache dns client sends dns request to remote dns server
func (r *Resolver) exchangeContextWithoutCache(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	// choose a nameserver and send dns request to it
	replyCh := make(chan *dns.Msg, 1)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	wg := sync.WaitGroup{}
	getReplyDnsMsg := func(nameserver string) {
		defer wg.Done()
		reply, _, err := r.client.ExchangeContext(ctx, req, nameserver)
		if err != nil {
			return
		}
		if reply.Rcode != dns.RcodeSuccess {
			return
		}
		select {
		case replyCh <- reply:
		default:
		}
	}

	for _, nameserver := range r.nameservers {
		wg.Add(1)
		go getReplyDnsMsg(nameserver)
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
		return nil, errors.New("can not get dns msg")
	}
}

func (r *Resolver) lookupHostsFile(req *dns.Msg) (*dns.Msg, error) {
	host := dnsutil.TrimDomain(req.Question[0].Name)
	ip := hostsutil.LookupIP(host)
	if !ip.IsValid() {
		return nil, errors.New("can not lookup ip")
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
