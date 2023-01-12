package dns

import (
	"net"
	"time"

	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/util"
	"github.com/miekg/dns"
)

var (
	// default dns nameservers
	DefaultDnsNameservers = []string{
		"114.114.114.114",
		"8.8.8.8",
	}
)

type DnsServer struct {
	Addr   string
	Port   int
	server *dns.Server
}

func NewDnsServer(addr string) *DnsServer {
	var port int
	_, p, _ := net.SplitHostPort(addr)
	if p == "" {
		port = 53
	} else {
		port = util.MustStringToInt(p)
	}
	s := &DnsServer{
		Addr: addr,
		Port: port,
	}

	// local dns server
	dnsServer := &dns.Server{
		Addr:         addr,
		Net:          "udp",
		UDPSize:      4096,
		Handler:      dns.HandlerFunc(s.serveDNS),
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
	}
	s.server = dnsServer
	return s
}

func (s *DnsServer) Start() error {
	return s.server.ListenAndServe()
}

func (s *DnsServer) Close() error {
	return s.server.Shutdown()
}

func (s *DnsServer) LocalAddress() string {
	return s.Addr
}

func (s *DnsServer) serveDNS(w dns.ResponseWriter, r *dns.Msg) {
	reply, err := resolver.DefaultResolver.Query(r)
	if err != nil {
		dns.HandleFailed(w, r)
	} else {
		w.WriteMsg(reply)
	}
}
