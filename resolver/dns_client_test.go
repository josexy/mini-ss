package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestDnsClient_ExchangeContext(t *testing.T) {
	nameservers := []struct {
		addr   string
		dnsNet string
	}{
		{"8.8.8.8:53", "udp"},
		{"8.8.8.8:53", "tcp"},
		{"dot.pub:853", "tcp-tls"},
		{"dns.alidns.com:853", "tcp-tls"},
		{"dns.tuna.tsinghua.edu.cn:8853", "tcp-tls"},
		{"https://doh.pub/dns-query", "https"},
		{"https://1.12.12.12/dns-query", "https"},
		{"https://120.53.53.53/dns-query", "https"},
		{"https://223.6.6.6/dns-query", "https"},
		{"https://dns.alidns.com/dns-query", "https"},
	}

	for _, nameserver := range nameservers {
		client := NewDnsClient(nameserver.dnsNet, nameserver.addr, 5*time.Second)
		req := new(dns.Msg)
		req.SetQuestion(dns.Fqdn("www.example.com"), dns.TypeA)
		req.RecursionDesired = true
		reply, err := client.ExchangeContext(context.Background(), req)
		assert.Nil(t, err)
		if reply != nil {
			t.Log(nameserver.dnsNet, nameserver.addr, reply.Answer)
		}
	}
}
