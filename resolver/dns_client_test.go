package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
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

	group := singleflight.Group{}
	for _, nameserver := range nameservers {
		client := NewDnsClient(nameserver.dnsNet, nameserver.addr, 5*time.Second)
		for i := 0; i < 5; i++ {
			go func() {
				timeout := time.Millisecond * time.Duration(i*50)
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				lookupCtx, lookupCancel := context.WithCancel(ctx)
				key := nameserver.dnsNet + nameserver.addr
				resCh := group.DoChan(key, func() (interface{}, error) {
					req := new(dns.Msg)
					req.SetQuestion(dns.Fqdn("www.example.com"), dns.TypeA)
					req.RecursionDesired = true
					reply, err := client.ExchangeContext(lookupCtx, req)
					return reply, err
				})
				select {
				case <-ctx.Done():
					t.Log(timeout, nameserver.dnsNet, nameserver.addr, ctx.Err())
					lookupCancel()
					return
				case res := <-resCh:
					dnsMsg, ok := res.Val.(*dns.Msg)
					if ok && dnsMsg != nil {
						t.Log(timeout, nameserver.dnsNet, nameserver.addr, dnsutil.MsgToAddrs(dnsMsg), res.Err, res.Shared)
					} else {
						t.Log(timeout, nameserver.dnsNet, nameserver.addr, res.Err, res.Shared)
					}
					lookupCancel()
				}
			}()
		}
	}
	time.Sleep(time.Second * 3)
}
