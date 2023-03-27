package enhancer

import (
	"net/netip"
	"runtime"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/netstackgo"
	"github.com/josexy/netstackgo/tun"
)

type EnhancerConfig struct {
	Tun     tun.TunConfig
	FakeDNS string
}

type Enhancer struct {
	nameserver netip.Addr
	config     EnhancerConfig
	nt         *netstackgo.TunNetstack
	fakeDns    *dns.DnsServer
	handler    *enhancerHandler
}

func NewEnhancer(config EnhancerConfig) *Enhancer {
	eh := &Enhancer{
		config:  config,
		nt:      netstackgo.New(config.Tun),
		fakeDns: dns.NewDnsServer(config.FakeDNS),
	}
	eh.handler = newEnhancerHandler(eh)
	return eh
}

func (eh *Enhancer) Start() (err error) {

	eh.nt.RegisterConnHandler(eh.handler)

	// start low-level gVisor netstack
	if err = eh.nt.Start(); err != nil {
		return
	}

	// start local fake dns server
	resolver.DefaultResolver.EnableEnhancerMode(eh.config.Tun.Addr)
	go func() {
		if err := eh.fakeDns.Start(); err != nil {
			logger.Logger.Warnf("%s", err.Error())
		}
	}()

	// set local dns server configuration
	if runtime.GOOS == "windows" {
		eh.nameserver = netip.MustParseAddr("127.0.0.1")
		dnsutil.SetLocalDnsServer(eh.config.Tun.Name)
	} else {
		ip := netip.MustParsePrefix(eh.config.Tun.Addr).Masked().Addr()
		ip = ip.Next().Next()
		eh.nameserver = ip
		dnsutil.SetLocalDnsServer(eh.nameserver.String())
	}

	logger.Logger.Info("create tun device",
		logx.String("name", eh.config.Tun.Name),
		logx.String("address", eh.config.Tun.Addr),
		logx.UInt32("mtu", eh.config.Tun.MTU))
	return
}

func (eh *Enhancer) Close() error {
	eh.fakeDns.Close()
	return eh.nt.Close()
}
