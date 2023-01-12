package tun

import (
	"context"
	"net"
	"net/netip"
	"strconv"

	"github.com/fatih/color"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/bufferpool"
	D "github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/socks/constant"
	"github.com/josexy/mini-ss/ss/ctxv"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/tun/core/adapter"
	"github.com/josexy/mini-ss/util/ordmap"
	"github.com/miekg/dns"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	DefaultMTU     = 1350
	maxSegmentSize = (1 << 16) - 1
	pool           = bufferpool.NewBufferPool(maxSegmentSize)
)

type IPRoute struct {
	// net.ParseCIDR() ip/netmask: 10.0.0.1/24
	Dest netip.Prefix
	// next hop router address
	Gateway netip.Addr
}

type TunConfig struct {
	Name string
	Addr string
	MTU  uint32
	// fake dns server
	FakeDnsAddr string
	// local nameserver
	LocalNameserver string
}

type ConnTuple struct {
	SrcIP   netip.Addr
	SrcPort int
	DstIP   netip.Addr
	DstPort int
}

func newConnTuple(id *stack.TransportEndpointID) *ConnTuple {
	srcIP, _ := netip.AddrFromSlice([]byte(id.RemoteAddress))
	dstIP, _ := netip.AddrFromSlice([]byte(id.LocalAddress))

	return &ConnTuple{
		SrcIP:   srcIP,
		SrcPort: int(id.RemotePort),
		DstIP:   dstIP,
		DstPort: int(id.LocalPort),
	}
}

func (t *ConnTuple) Src() string {
	return net.JoinHostPort(t.SrcIP.String(), strconv.FormatUint(uint64(t.SrcPort), 10))
}

func (t *ConnTuple) Dst() string {
	return net.JoinHostPort(t.DstIP.String(), strconv.FormatUint(uint64(t.DstPort), 10))
}

func (t *ConnTuple) Addr() netip.AddrPort {
	return netip.AddrPortFrom(t.DstIP, uint16(t.DstPort))
}

type tunTransportHandler struct {
	relayers ordmap.OrderedMap
	ruler    *D.Ruler
	tunStack *TunStack
	tcpQueue chan adapter.TCPConn
	udpQueue chan adapter.UDPConn
	closeCh  chan struct{}
	adapter.TransportHandler
}

func newTunTransportHandler(ctx context.Context, tunStack *TunStack) *tunTransportHandler {
	pv := ctx.Value(ctxv.SSLocalContextKey).(*ctxv.ContextPassValue)
	handler := &tunTransportHandler{
		tunStack: tunStack,
		ruler:    pv.R,
		tcpQueue: make(chan adapter.TCPConn, 128),
		udpQueue: make(chan adapter.UDPConn, 128),
		closeCh:  make(chan struct{}),
	}
	pv.MAP.Range(func(name, value any) bool {
		v := value.(ctxv.V)
		relayer := transport.DstAddrRelayer{
			DstAddr:          v.Addr,
			TCPRelayer:       transport.NewTCPRelayer(constant.TCPSSLocalToSSServer, v.Type, v.Options, nil, v.TcpConnBound),
			TCPDirectRelayer: transport.NewTCPDirectRelayer(),
			// UDP relay only support default transport type
			UDPRelayer:       transport.NewUDPRelayer(constant.UDPTunServerToSSServer, transport.Default, nil, v.UdpConnBound),
			UDPDirectRelayer: transport.NewUDPDirectRelayer(),
		}
		handler.relayers.Store(name, relayer)
		return true
	})
	if pv.MAP.Size() == 0 && pv.R.RuleMode == D.Direct {
		handler.relayers.Store("", transport.DstAddrRelayer{
			TCPDirectRelayer: transport.NewTCPDirectRelayer(),
			UDPDirectRelayer: transport.NewUDPDirectRelayer(),
		})
	}
	handler.TransportHandler = handler
	return handler
}

func (h *tunTransportHandler) Go() {
	go func() {
		defer func() { recover() }()
		for {
			select {
			case conn := <-h.tcpQueue:
				go h.handleTCPConn(conn)
			case conn := <-h.udpQueue:
				go h.handleUDPConn(conn)
			case <-h.closeCh:
				return
			}
		}
	}()
}

func (h *tunTransportHandler) Finish() {
	h.closeCh <- struct{}{}
}

func (h *tunTransportHandler) HandleTCP(conn adapter.TCPConn) {
	h.tcpQueue <- conn
}

func (h *tunTransportHandler) HandleUDP(conn adapter.UDPConn) {
	h.udpQueue <- conn
}

func (h *tunTransportHandler) handleTCPConn(conn adapter.TCPConn) {
	defer conn.Close()

	connTuple := newConnTuple(conn.ID())

	// DstIP: fakeip/realip
	record := resolver.DefaultResolver.FindByIP(connTuple.DstIP)

	dstAddr := connTuple.Addr()

	host := connTuple.DstIP.String()
	if record != nil {
		host = record.Domain
	}
	// match domain name and real ip address
	if h.ruler.Match(&host) {
		return
	}

	raddr := dstAddr.String()
	mr := h.ruler.MatcherResult()
	// if record != nil, it means dst address is fake ip
	// for DIRECT mdoe, resolve ip address locally
	// for GLOBAL/MATCH mode, resolve ip address remotely
	if record != nil {
		if mr.RuleMode == D.Direct {
			if !record.RealIP.IsValid() {
				record.RealIP = resolver.DefaultResolver.ResolveQuery(record.Query)
				if !record.RealIP.IsValid() {
					return
				}
			}
			dstAddr = netip.AddrPortFrom(record.RealIP, uint16(connTuple.DstPort))
			// ip:port
			raddr = dstAddr.String()
		} else {
			// host:port
			// the host may be a domain name or an ip address
			raddr = net.JoinHostPort(host, strconv.Itoa(connTuple.DstPort))
		}
	}

	logx.Debug("[%s]: %s -> %s", color.RedString("tun-tcp"), color.GreenString(connTuple.Src()), color.YellowString(raddr))

	tcpTracker := statistic.NewTcpTracker(conn, connTuple.Src(), connTuple.Dst(),
		statistic.LazyContext{
			Host:     raddr,
			RuleMode: mr.RuleMode.String(),
			RuleType: string(mr.RuleType),
			Proxy:    mr.Proxy,
		},
		statistic.DefaultManager)
	h.ruler.Select(&h.relayers, tcpTracker, raddr)
}

func (h *tunTransportHandler) handleUDPConn(conn adapter.UDPConn) {
	defer conn.Close()

	connTuple := newConnTuple(conn.ID())
	// the local DNS server handles the DNS request or the remote server handles the DNS request
	if connTuple.DstPort == h.tunStack.fakeDns.Port {
		if connTuple.DstIP.String() == h.tunStack.tunCfg.LocalNameserver {
			h.relayDnsRequest(conn)
		}
		return
	}

	addr := connTuple.DstIP.String()
	// match the destination ip address by GeoIP and IP-CIDR rule
	if h.ruler.Match(&addr) {
		return
	}

	// the raddr may be host:port or ip:port
	raddr := net.JoinHostPort(addr, strconv.Itoa(connTuple.DstPort))

	logx.Debug("[%s]: %s -> %s", color.CyanString("tun-udp"), color.GreenString(connTuple.Src()), color.YellowString(raddr))

	udpTracker := statistic.NewUdpTracker(conn, connTuple.Src(), raddr, statistic.LazyContext{}, statistic.DefaultManager)
	for {
		if err := h.ruler.SelectOne(&h.relayers, udpTracker, raddr); err != nil {
			return
		}
	}
}

func (h *tunTransportHandler) relayDnsRequest(conn net.PacketConn) error {
	b := pool.Get()
	defer pool.Put(b)
	n, srcAddr, err := conn.ReadFrom(*b)
	if err != nil {
		return err
	}
	req := new(dns.Msg)
	if err = req.Unpack((*b)[:n]); err != nil {
		return err
	}

	// for tun mode, we can return fake ip address directly
	// and don't resolve the domain name as far as possible
	reply, err := resolver.DefaultResolver.Query(req)
	if err != nil {
		return err
	}
	data, _ := reply.PackBuffer(*b)
	conn.WriteTo(data, srcAddr)
	return nil
}
