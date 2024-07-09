package resolver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/hostsutil"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
)

var (
	errCannotLookupIPFromHostsFile = errors.New("cannot lookup ip from local hosts file")
	errCannotLookupIPv4v6          = errors.New("cannot lookup ipv4 and ipv6 from host")
	errDnsExchangedFailed          = errors.New("dns exchanged failed")
)

var DefaultResolver *Resolver

type nameserverExt struct {
	addr   string
	dnsNet string
}

type Resolver struct {
	*fakeIPResolver
	// UDP/TCP/DoT/DoH
	nameservers    []nameserverExt
	clients        map[string]*DnsClient
	lookupGroup    singleflight.Group
	lookupHostPref bool
}

func parseNameserver(nameservers []string) []nameserverExt {
	addPrefix := func(nameserver string) string {
		if strings.Contains(nameserver, "://") {
			return nameserver
		}
		if ip, err := netip.ParseAddr(nameserver); err != nil {
			return "udp://" + nameserver
		} else {
			if ip.Is4() {
				return "udp://" + nameserver
			} else {
				return "udp://[" + nameserver + "]"
			}
		}
	}
	formatNameserver := func(hostport, defaultPort string) (string, error) {
		host, port, err := net.SplitHostPort(hostport)
		if err != nil {
			if !strings.Contains(err.Error(), "missing port in address") {
				return "", err
			}
			hostport = hostport + ":" + defaultPort
			if host, port, err = net.SplitHostPort(hostport); err != nil {
				return "", err
			}
		}
		return net.JoinHostPort(host, port), nil
	}
	var list []nameserverExt
	for _, ns := range nameservers {
		ns = addPrefix(ns)
		urlres, err := url.Parse(ns)
		if err != nil {
			logger.Logger.ErrorBy(err)
			continue
		}
		var addr, dnsNet string
		switch urlres.Scheme {
		case "udp":
			dnsNet = "udp"
			addr, err = formatNameserver(urlres.Host, "53") // DNS over UDP
		case "tcp":
			dnsNet = "tcp"
			addr, err = formatNameserver(urlres.Host, "53") // DNS over TCP
		case "tls":
			dnsNet = "tcp-tls"
			addr, err = formatNameserver(urlres.Host, "853") // DNS over TLS
		case "https":
			dnsNet = "https"
			addr, err = formatNameserver(urlres.Host, "443") // DNS over HTTPS
			if err == nil {
				urlInfo := url.URL{Scheme: "https", Host: addr, Path: urlres.Path, User: urlres.User}
				addr = urlInfo.String()
			}
		default:
			logger.Logger.Errorf("unsupported dns scheme: %s", urlres.Scheme)
			continue
		}
		if err != nil {
			logger.Logger.ErrorBy(err)
			continue
		}
		list = append(list, nameserverExt{
			addr:   addr,
			dnsNet: dnsNet,
		})
	}
	return list
}

func NewDnsResolver(nameservers []string, lookupHostsFile bool) *Resolver {
	nameserver := parseNameserver(nameservers)
	// default system config dns
	fallback := parseNameserver(dnsutil.GetLocalDnsList())
	if len(fallback) == 0 {
		logger.Logger.Warn("read system dns config empty")
	}

	resolver := &Resolver{
		clients:        make(map[string]*DnsClient),
		nameservers:    append(nameserver, fallback...),
		lookupHostPref: lookupHostsFile,
	}

	for _, ns := range resolver.nameservers {
		logger.Logger.Infof("dns nameserver: type: %s, addr: %s", ns.dnsNet, ns.addr)
		resolver.clients[ns.dnsNet+":"+ns.addr] = NewDnsClient(ns.dnsNet, ns.addr, time.Second*5)
	}
	return resolver
}

func (r *Resolver) IsEnhancerMode() bool {
	return r.fakeIPResolver != nil
}

func (r *Resolver) EnableEnhancerMode(tunCIDR netip.Prefix) (err error) {
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
	if r.lookupHostPref {
		ip := hostsutil.LookupIP(host)
		if ip.IsValid() {
			return []netip.Addr{ip}, nil
		}
	}
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
	if err == nil && len(ips) > 0 {
		return ips, nil
	}
	ips, ok := <-ipsCh
	if !ok {
		return nil, errCannotLookupIPv4v6
	}
	return ips, nil
}

func (r *Resolver) ResolveTCPAddr(ctx context.Context, addr string) (*net.TCPAddr, error) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ip := r.LookupHost(ctx, host)
	if !ip.IsValid() {
		return nil, fmt.Errorf("can not lookup address: %s", addr)
	}
	port, _ := strconv.ParseUint(p, 10, 16)
	return &net.TCPAddr{
		IP:   ip.AsSlice(),
		Port: int(port),
	}, nil
}

func (r *Resolver) ResolveUDPAddr(ctx context.Context, addr string) (*net.UDPAddr, error) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ip := r.LookupHost(ctx, host)
	if !ip.IsValid() {
		return nil, fmt.Errorf("can not lookup address: %s", addr)
	}
	port, _ := strconv.ParseUint(p, 10, 16)
	return &net.UDPAddr{
		IP:   ip.AsSlice(),
		Port: int(port),
	}, nil
}

func (r *Resolver) lookupIP(ctx context.Context, host string, dnsType uint16) ([]netip.Addr, error) {
	req := &dns.Msg{}
	req.SetQuestion(dns.Fqdn(host), dnsType)
	req.RecursionDesired = true

	reply, err := r.lookupIPWithMsg(ctx, req)
	if err != nil {
		return nil, err
	}
	addrs := dnsutil.MsgToAddrs(reply)
	logger.Logger.Debug("lookupIP succeed",
		logx.String("query", host),
		logx.String("type", dns.TypeToString[dnsType]),
		logx.String("rcode", dns.RcodeToString[reply.Rcode]),
		logx.Slice3("ips", addrs),
	)
	return addrs, nil
}

func (r *Resolver) lookupIPWithMsg(ctx context.Context, req *dns.Msg) (*dns.Msg, error) {
	lookupCtx, lookupCancel := context.WithCancel(ctx)

	ch := r.lookupGroup.DoChan(req.Question[0].String(), func() (interface{}, error) {
		return r.exchangeContext(lookupCtx, req)
	})

	select {
	case <-ctx.Done():
		lookupCancel()
		return nil, ctx.Err()
	case r := <-ch:
		lookupCancel()
		if r.Err != nil {
			return nil, r.Err
		}
		if reply, ok := r.Val.(*dns.Msg); ok {
			if r.Shared {
				reply = reply.Copy()
			}
			return reply, nil
		} else {
			return nil, errors.New("invalid dns lookup msg")
		}
	}
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

	getReplyDnsMsg := func(ctx context.Context, key string) {
		defer wg.Done()
		reply, err := r.clients[key].ExchangeContext(ctx, req)
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
		key := nameserver.dnsNet + ":" + nameserver.addr
		go getReplyDnsMsg(ctx, key)
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
		if r.matchDomainFilter(req) {
			logger.Logger.Debugf("domain filter matched for %s", req.Question[0].Name)
			reply, err = r.lookupIPWithMsg(context.Background(), req)
			return
		}
		// ipv4 dns query, return fake ip address
		reply, err = r.fakeIPResolver.query(req)
	} else {
		// return an empty response for ipv6
		reply, err = &dns.Msg{}, nil
		reply.SetReply(req)
	}
	return
}

func (r *Resolver) matchDomainFilter(req *dns.Msg) bool {
	if len(DefaultDomainFilter) == 0 {
		return false
	}
	domain := dnsutil.TrimDomain(req.Question[0].Name)
	// TODO: support wildcard domain match?
	for _, name := range DefaultDomainFilter {
		if domain == name {
			return true
		}
	}
	return false
}
