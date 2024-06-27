package main

import (
	"fmt"
	"time"

	"github.com/josexy/mini-ss/util/cache"
	"github.com/miekg/dns"
)

func main() {
	dnsCache := cache.NewCache[string, *dns.Msg](
		cache.WithBackgroundCheckCache(),
		cache.WithDeleteExpiredCacheOnGet(),
		cache.WithGoTimeNow(),
		cache.WithMaxSize(100),
		cache.WithInterval(time.Second*5),
		cache.WithExpiration(time.Second*10),
		cache.WithEvictCallback(func(a1, a2 any) {
			fmt.Println("evict dns cache", a1)
		}),
	)
	dnsClient := &dns.Client{}
	server := &dns.Server{
		Net:  "udp",
		Addr: ":5353",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
			domain := m.Question[0].Name
			qtype := dns.Type(m.Question[0].Qtype)
			fmt.Println(domain, qtype)

			key := domain + ":" + qtype.String()
			if reply, err := dnsCache.Get(key); err == nil {
				reply.SetReply(m)
				w.WriteMsg(reply)
				return
			}

			reply, _, err := dnsClient.Exchange(m, "8.8.8.8:53")
			if err != nil {
				r := new(dns.Msg)
				r.SetRcode(m, dns.RcodeServerFailure)
				w.WriteMsg(r)
				return
			}
			w.WriteMsg(reply)
			dnsCache.Set(key, reply.Copy())
		}),
	}
	server.ListenAndServe()
}
