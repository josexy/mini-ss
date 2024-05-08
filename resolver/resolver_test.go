package resolver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/logger"
)

func TestDnsResolver(t *testing.T) {
	nameservers := []string{
		"udp://8.8.8.8:53",
		"tcp://8.8.8.8:53",
		"tls://dot.pub:853",
		"tls://dns.alidns.com:853",
		"tls://dns.tuna.tsinghua.edu.cn:8853",
		"https://doh.pub/dns-query",
		"https://1.12.12.12/dns-query",
		"https://120.53.53.53/dns-query",
		"https://223.6.6.6/dns-query",
		"https://dns.alidns.com/dns-query",
	}
	logger.Logger = logger.LogContext.Copy().WithCaller(false, true, false, true).BuildConsoleLogger(logx.LevelTrace)
	r := NewDnsResolver(nameservers)

	for _, ns := range nameservers {
		for i := 0; i < 2; i++ {
			go func() {
				timeout := time.Millisecond * time.Duration(2000)
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()
				domain := fmt.Sprintf("www.example%d.com", i)
				ips, err := r.LookupIP(ctx, domain)
				t.Log(ns, domain, ips, err)
			}()
		}
	}
	time.Sleep(time.Second * 5)
}
