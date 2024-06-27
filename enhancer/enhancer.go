package enhancer

import (
	"net/netip"
	"sync/atomic"

	tun "github.com/josexy/cropstun"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/util/logger"
)

type EnhancerConfig struct {
	Tun            tun.Options
	FakeDNS        string
	DisableRewrite bool
	DnsHijack      []netip.AddrPort
}

type Enhancer struct {
	dnsAddress netip.Addr
	config     EnhancerConfig
	stack      tun.Stack
	fakeDns    *resolver.DnsServer
	handler    *enhancerHandler
	running    atomic.Bool
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
	if eh.running.Load() {
		return
	}

	// init fake ip pool and cache
	if err = resolver.DefaultResolver.EnableEnhancerMode(eh.config.Tun.Inet4Address[0]); err != nil {
		return
	}

	eh.config.Tun.Inet4Address[0] = resolver.DefaultResolver.GetAllocatedTunPrefix()
	eh.config.Tun.IPRoute2TableIndex = 10086
	eh.config.Tun.IPRoute2RuleIndex = 5000

	var tunDevice tun.Tun
	if tunDevice, err = tun.NewTunDevice(nil, &eh.config.Tun); err != nil {
		return
	}

	if eh.stack, err = tun.NewStack(tun.StackOptions{
		Tun:        tunDevice,
		TunOptions: &eh.config.Tun,
		Handler:    eh.handler,
	}); err != nil {
		return
	}

	if err = eh.stack.Start(); err != nil {
		return
	}

	go func() {
		if err := eh.fakeDns.Start(); err != nil {
			logger.Logger.ErrorBy(err)
		}
	}()

	eh.dnsAddress = resolver.DefaultResolver.GetAllocatedDnsIP()

	if !eh.config.DisableRewrite && eh.dnsAddress.IsValid() {
		logger.Logger.Infof("setup dns address: %s", eh.dnsAddress.String())
		eh.stack.TunDevice().SetupDNS([]netip.Addr{eh.dnsAddress})
	}

	logger.Logger.Info("create tun device",
		logx.String("name", eh.config.Tun.Name),
		logx.String("address", eh.config.Tun.Inet4Address[0].String()),
		logx.UInt32("mtu", eh.config.Tun.MTU),
		logx.Slice3("dns-hijack", eh.config.DnsHijack),
		logx.Bool("auto-route", eh.config.Tun.AutoRoute))

	eh.running.Store(true)
	return
}

func (eh *Enhancer) Close() error {
	if !eh.running.Load() {
		return nil
	}
	if !eh.config.DisableRewrite {
		eh.stack.TunDevice().TeardownDNS()
	}
	eh.fakeDns.Close()
	err := eh.stack.Close()
	eh.running.Store(false)
	return err
}
