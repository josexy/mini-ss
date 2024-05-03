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
	Tun            tun.TunConfig
	FakeDNS        string
	DisableRewrite bool
	tunCidr        netip.Prefix
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
	if err = resolver.DefaultResolver.EnableEnhancerMode(eh.config.Tun.Addr); err != nil {
		return
	}

	go func() {
		if err := eh.fakeDns.Start(); err != nil {
			logger.Logger.Warnf("%s", err.Error())
		}
	}()

	eh.config.tunCidr = netip.MustParsePrefix(eh.config.Tun.Addr)
	ip := eh.config.tunCidr.Masked().Addr()
	ip = ip.Next().Next()
	eh.nameserver = ip

	// set local dns server configuration
	if runtime.GOOS != "windows" && !eh.config.DisableRewrite {
		logger.Logger.Infof("rewrite fake dns server to system config file: %s", eh.nameserver)
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
