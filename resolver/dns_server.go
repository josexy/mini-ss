package resolver

import (
	"net"
	"strconv"
	"time"

	"github.com/josexy/mini-ss/util/logger"
	"github.com/miekg/dns"
)

var (
	// DefaultDnsNameservers default dns nameservers
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
	_, p, _ := net.SplitHostPort(addr)
	if p == "" {
		p = "53"
	}
	port, _ := strconv.ParseUint(p, 10, 16)
	s := &DnsServer{Addr: addr, Port: int(port)}
	// local dns server
	s.server = &dns.Server{
		Addr:         addr,
		Net:          "udp",
		UDPSize:      4096,
		Handler:      dns.HandlerFunc(s.serveDNS),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}
	return s
}

func (s *DnsServer) Start() error {
	logger.Logger.Infof("start dns server on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *DnsServer) Close() error {
	return s.server.Shutdown()
}

func (s *DnsServer) LocalAddress() string {
	return s.Addr
}

func (s *DnsServer) serveDNS(w dns.ResponseWriter, r *dns.Msg) {
	reply, err := DefaultResolver.Query(r)
	if err != nil {
		logger.Logger.ErrorBy(err)
		dns.HandleFailed(w, r)
	} else {
		w.WriteMsg(reply)
	}
}
