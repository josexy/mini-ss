package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/miekg/dns"
)

var domain string
var dnsResolverIP string

// go run main.go query 192.168.1.1:53 www.google.com A
// go run main.go http 192.168.1.1:53 http://www.baidu.com

func main() {
	switch os.Args[1] {
	case "http":
		httpMain()
	case "query":
		queryMain()
	}
}

func httpMain() {
	dnsResolverIP = "127.0.0.1:53"
	domain = "http://myip.ipip.net"

	if len(os.Args) > 2 {
		dnsResolverIP = os.Args[2]
	}
	if len(os.Args) > 3 {
		domain = os.Args[3]
	}

	var (
		dnsResolverProto     = "udp"
		dnsResolverTimeoutMs = 2000
	)

	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
				}
				return d.DialContext(ctx, dnsResolverProto, address)
			},
		},
	}

	http.DefaultTransport.(*http.Transport).DialContext = dialer.DialContext
	httpClient := http.DefaultClient

	resp, err := httpClient.Get(domain)

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	log.Println(string(body))
}

func queryMain() {
	c := &dns.Client{
		Net: "udp",
	}
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(os.Args[3]), dns.StringToType[os.Args[4]])
	m.RecursionDesired = true

	r, _, err := c.Exchange(m, os.Args[2])
	if err != nil {
		panic(err)
	}
	if r.Rcode != dns.RcodeSuccess {
		return
	}
	for _, x := range r.Answer {
		log.Println(x.String())
	}
}
