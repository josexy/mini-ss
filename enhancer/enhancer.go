package enhancer

import (
	"net/netip"
	"runtime"

	"github.com/josexy/logx"
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
}

type Enhancer struct {
	nameserver netip.Addr
	config     EnhancerConfig
	nt         *netstackgo.TunNetstack
	fakeDns    *resolver.DnsServer
	handler    *enhancerHandler
}

func NewEnhancer(config EnhancerConfig) *Enhancer {
	eh := &Enhancer{
		config:  config,
		fakeDns: resolver.NewDnsServer(config.FakeDNS),
	}
	eh.handler = newEnhancerHandler(eh)
	return eh
}

func (eh *Enhancer) Start() (err error) {
	// init fake ip pool and cache
	if err = resolver.DefaultResolver.EnableEnhancerMode(eh.config.Tun.Addr); err != nil {
		return
	}

	eh.config.Tun.Addr = resolver.DefaultResolver.GetAllocatedTunPrefix().String()
	eh.nt = netstackgo.New(eh.config.Tun)
	eh.nt.RegisterConnHandler(eh.handler)

	// start low-level gVisor netstack
	if err = eh.nt.Start(); err != nil {
		return
	}

	go func() {
		if err := eh.fakeDns.Start(); err != nil {
			logger.Logger.Warnf("%s", err.Error())
		}
	}()

	eh.nameserver = resolver.DefaultResolver.GetAllocatedDnsIP()

	// set local dns server configuration
	if runtime.GOOS != "windows" && !eh.config.DisableRewrite && eh.nameserver.IsValid() {
		logger.Logger.Infof("rewrite dns fake ip %s to system config file", eh.nameserver)
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
