package enhancer

import (
	"net/netip"
	"runtime"

	tun "github.com/josexy/cropstun"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/logger"
)

type EnhancerConfig struct {
	Tun            tun.Options
	FakeDNS        string
	DisableRewrite bool
	DnsHijack      []netip.AddrPort
}

type Enhancer struct {
	nameserver netip.Addr
	config     EnhancerConfig
	stack      tun.Stack
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
	if err = resolver.DefaultResolver.EnableEnhancerMode(eh.config.Tun.Inet4Address[0]); err != nil {
		return
	}

	var tunDevice tun.Tun
	if tunDevice, err = tun.NewTunDevice(nil, &eh.config.Tun); err != nil {
		return
	}

	eh.config.Tun.Inet4Address[0] = resolver.DefaultResolver.GetAllocatedTunPrefix()
	if eh.stack, err = tun.NewStack(tun.StackOptions{
		Tun:        tunDevice,
		TunOptions: &eh.config.Tun,
		Handler:    eh.handler,
	}); err != nil {
		return
	}

	// start low-level gVisor netstack
	if err = eh.stack.Start(); err != nil {
		return
	}

	go func() {
		if err := eh.fakeDns.Start(); err != nil {
			logger.Logger.Warnf("%s", err.Error())
		}
	}()

	eh.nameserver = resolver.DefaultResolver.GetAllocatedDnsIP()

	// set local dns server configuration
	if runtime.GOOS == "darwin" && !eh.config.DisableRewrite && eh.nameserver.IsValid() {
		logger.Logger.Infof("rewrite dns fake ip %s to system config file", eh.nameserver)
		dnsutil.SetLocalDnsServer(eh.nameserver.String())
	}

	logger.Logger.Info("create tun device",
		logx.String("name", eh.config.Tun.Name),
		logx.String("address", eh.config.Tun.Inet4Address[0].String()),
		logx.UInt32("mtu", eh.config.Tun.MTU))
	return
}

func (eh *Enhancer) Close() error {
	eh.fakeDns.Close()
	return eh.stack.Close()
}
