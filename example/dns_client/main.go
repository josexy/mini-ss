package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// go run main.go query 8.8.8.8:53 www.google.com A
// go run main.go http 8.8.8.8:53 http://www.baidu.com

func main() {
	switch os.Args[1] {
	case "http":
		httpMain()
	case "query":
		queryMain()
	}
}

func httpMain() {
	dnsResolverIP := "127.0.0.1:53"
	domain := "http://myip.ipip.net"

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
				return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
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
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(os.Args[3]), dns.StringToType[os.Args[4]])
	m.RecursionDesired = true

	dnsAddr := os.Args[2]
	protocol := "udp"
	if strings.Contains(dnsAddr, "://") {
		parts := strings.Split(dnsAddr, "://")
		protocol, dnsAddr = parts[0], parts[1]
	}
	log.Println(protocol, dnsAddr)
	c := &dns.Client{
		Net:     protocol,
		UDPSize: 4096,
	}
	r, _, err := c.Exchange(m, dnsAddr)
	if err != nil {
		panic(err)
	}
	for _, x := range r.Answer {
		log.Println(x.String())
	}
}
