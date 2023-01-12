package tun

import (
	"context"
	"net"
	"net/netip"
	"runtime"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/tun/core/device"
	"github.com/josexy/mini-ss/tun/core/device/tun"
	"github.com/josexy/mini-ss/tun/core/option"
	"github.com/josexy/mini-ss/tun/internal"
	"github.com/josexy/mini-ss/util/dnsutil"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

var defaultGlobalRouteCIDRS = []string{
	"1.0.0.0/8",
	"2.0.0.0/7",
	"4.0.0.0/6",
	"8.0.0.0/5",
	"16.0.0.0/4",
	"32.0.0.0/3",
	"64.0.0.0/2",
	"128.0.0.0/1",
}

type TunStack struct {
	s         *stack.Stack
	tunDevice device.Device
	tunCfg    TunConfig
	handler   *tunTransportHandler
	fakeDns   *dns.DnsServer
}

func NewTunStack(ctx context.Context, tunCfg TunConfig) *TunStack {
	ts := &TunStack{
		tunCfg: tunCfg,
	}
	ts.fakeDns = dns.NewDnsServer(ts.tunCfg.FakeDnsAddr)
	resolver.DefaultResolver.EnableFakeIP(tunCfg.Addr)

	ts.handler = newTunTransportHandler(ctx, ts)
	return ts
}

func (ts *TunStack) openTun() (err error) {
	ts.tunDevice, err = tun.Open(ts.tunCfg.Name, uint32(ts.tunCfg.MTU))
	if err != nil {
		return err
	}
	return
}

func (ts *TunStack) Start() (err error) {
	// create local tun device
	if err = ts.openTun(); err != nil {
		return
	}

	if err = setTunIPAddress(ts.tunCfg.Name, ts.tunCfg.Addr, int(ts.tunCfg.MTU)); err != nil {
		return
	}

	tunSubnet := netip.MustParsePrefix(ts.tunCfg.Addr)
	var routes []IPRoute

	for _, cidr := range defaultGlobalRouteCIDRS {
		routes = append(routes, IPRoute{
			Dest:    netip.MustParsePrefix(cidr),
			Gateway: tunSubnet.Addr(), // redirect to tun device
		})
	}

	if err = addTunNetRoutes(ts.tunCfg.Name, routes); err != nil {
		logx.ErrorBy(err)
		return
	}

	ts.handler.Go()

	// start fake dns server
	go func() {
		if err := ts.fakeDns.Start(); err != nil {
			logx.Warn("%s", err.Error())
		}
	}()

	// initialize gVisor netstack
	if err = ts.createStack(); err != nil {
		logx.FatalBy(err)
		return
	}

	if runtime.GOOS == "windows" {
		ts.tunCfg.LocalNameserver = "127.0.0.1"
		dnsutil.SetLocalDnsServer(ts.tunCfg.Name)
	} else {
		ip, _, _ := net.ParseCIDR(ts.tunCfg.Addr)
		ip = ip.To4()
		ip = ip.Mask(ip.DefaultMask())
		ip[3] += 2
		ts.tunCfg.LocalNameserver = ip.String()
		dnsutil.SetLocalDnsServer(ts.tunCfg.LocalNameserver)
	}

	logx.Info("start tun device name:%s, mtu:%d", ts.tunDevice.Name(), ts.tunDevice.MTU())
	return
}

func (ts *TunStack) Close() error {
	ts.fakeDns.Close()
	err := ts.tunDevice.Close()
	ts.s.Close()
	ts.s.Wait()
	ts.handler.Finish()
	return err
}

func (ts *TunStack) createStack() error {
	ts.s = stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})

	nicID := tcpip.NICID(ts.s.UniqueID())

	opts := []option.Option{option.WithDefault()}

	opts = append(opts,
		// Important: We must initiate transport protocol handlers
		// before creating NIC, otherwise NIC would dispatch packets
		// to stack and cause race condition.
		// Initiate transport protocol (TCP/UDP) with given handler.
		internal.WithTCPHandler(ts.handler.HandleTCP),
		internal.WithUDPHandler(ts.handler.HandleUDP),

		// Create stack NIC and then bind link endpoint to it.
		internal.WithCreatingNIC(nicID, ts.tunDevice),

		// In the past we did s.AddAddressRange to assign 0.0.0.0/0
		// onto the interface. We need that to be able to terminate
		// all the incoming connections - to any ip. AddressRange API
		// has been removed and the suggested workaround is to use
		// Promiscuous mode. https://github.com/google/gvisor/issues/3876
		//
		// Ref: https://github.com/cloudflare/slirpnetstack/blob/master/stack.go
		internal.WithPromiscuousMode(nicID, internal.NicPromiscuousModeEnabled),

		// Enable spoofing if a stack may send packets from unowned
		// addresses. This change required changes to some netgophers
		// since previously, promiscuous mode was enough to let the
		// netstack respond to all incoming packets regardless of the
		// packet's destination address. Now that a stack.Route is not
		// held for each incoming packet, finding a route may fail with
		// local addresses we don't own but accepted packets for while
		// in promiscuous mode. Since we also want to be able to send
		// from any address (in response the received promiscuous mode
		// packets), we need to enable spoofing.
		//
		// Ref: https://github.com/google/gvisor/commit/8c0701462a84ff77e602f1626aec49479c308127
		internal.WithSpoofing(nicID, internal.NicSpoofingEnabled),

		// Add default route table for IPv4 and IPv6. This will handle
		// all incoming ICMP packets.
		internal.WithRouteTable(nicID),
	)

	for _, opt := range opts {
		if err := opt(ts.s); err != nil {
			return err
		}
	}
	return nil
}
