package enhancer

import (
	"io"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/netstackgo"
	"github.com/miekg/dns"
)

const (
	DefaultMTU    = 1350
	dnsMsgTimeout = time.Second * 5
)

var stackTraceBufferPool = bufferpool.NewBufferPool(4096)

type enhancerHandler struct {
	owner *Enhancer
	pool  *bufferpool.BufferPool
}

func newEnhancerHandler(eh *Enhancer) *enhancerHandler {
	return &enhancerHandler{
		owner: eh,
		pool:  bufferpool.NewBufferPool(4096 * 2),
	}
}

func (handler *enhancerHandler) HandleTCPConn(info netstackgo.ConnTuple, conn net.Conn) {
	// the target address(info.DstAddr.Addr()) may be a fake ip address or real ip address
	// for example `curl www.google.com` or `curl 74.125.24.103:80`

	// note: The following request methods will not handle the remote request address,
	// but will be handled by the proxy node server
	// `curl --socks5 127.0.0.1:10088 74.125.24.103:80`
	// `curl --proxy 127.0.0.1:10088 74.125.24.103:80`
	// `curl --socks5 127.0.0.1:10088 www.google.com`
	// `curl --proxy 127.0.0.1:10088 www.google.com`

	defer func() {
		if err := recover(); err != nil {
			buf := stackTraceBufferPool.Get()
			n := runtime.Stack(*buf, false)
			logger.Logger.Error("tun connection recovery",
				logx.Error("err", err.(error)),
				logx.String("stackbuf", string((*buf)[:n])),
			)
			stackTraceBufferPool.Put(buf)
		}
	}()

	if handler.isNeedHijackDNS(info.DstAddr) {
		if err := handler.hijackDNSForTCP(conn); err != nil {
			logger.Logger.ErrorBy(err)
		}
		return
	}

	var remote string
	dstIp := info.DstAddr.Addr()
	if resolver.DefaultResolver.IsFakeIP(dstIp) {
		if fakeDnsRecord, err := resolver.DefaultResolver.FindByIP(dstIp); err == nil {
			remote = fakeDnsRecord.Domain
			logger.Logger.Debug("find the domain from fake ip",
				logx.String("fakeip", dstIp.String()),
				logx.UInt16("port", info.DstAddr.Port()),
				logx.String("domain", fakeDnsRecord.Domain))
		} else {
			// fake ip/record not found or expired
			logger.Logger.ErrorBy(err)
			return
		}
	} else {
		remote = dstIp.String()
	}

	if !rule.MatchRuler.Match(&remote) {
		return
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	remoteAddr := net.JoinHostPort(remote, strconv.FormatUint(uint64(info.DstAddr.Port()), 10))

	logger.Logger.Info("tcp-tun",
		logx.String("src", info.Src()),
		logx.String("dst", info.Dst()),
		logx.String("remote", remoteAddr),
	)

	if statistic.EnableStatistic {
		tcpTracker := statistic.NewTCPTracker(conn, statistic.Context{
			Src:     info.Src(),
			Dst:     remoteAddr,
			Type:    "TCP-TUN",
			Network: "TCP",
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
			Proxy:   proxy,
		})
		defer statistic.DefaultManager.Remove(tcpTracker)
		conn = tcpTracker
	}
	if err := selector.ProxySelector.Select(proxy).Invoke(conn, remoteAddr); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (handler *enhancerHandler) HandleUDPConn(info netstackgo.ConnTuple, conn net.PacketConn) {

	defer func() {
		if err := recover(); err != nil {
			buf := stackTraceBufferPool.Get()
			n := runtime.Stack(*buf, false)
			logger.Logger.Error("tun connection recovery",
				logx.Error("err", err.(error)),
				logx.String("stackbuf", string((*buf)[:n])),
			)
			stackTraceBufferPool.Put(buf)
		}
	}()

	if handler.isNeedHijackDNS(info.DstAddr) {
		if err := handler.hijackDNSForUDP(conn); err != nil {
			logger.Logger.ErrorBy(err)
		}
		return
	}

	// discard udp fake ip
	if resolver.DefaultResolver.IsFakeIP(info.DstAddr.Addr()) {
		return
	}

	// for UDP request matching, support GeoIP and IP-CIDR
	remote := info.DstAddr.Addr().String()
	if !rule.MatchRuler.Match(&remote) {
		return
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	logger.Logger.Info("udp-tun", logx.String("src", info.Src()), logx.String("dst", info.Dst()))

	if statistic.EnableStatistic {
		udpTracker := statistic.NewUDPTracker(conn, statistic.Context{
			Src:     info.Src(),
			Dst:     info.Dst(),
			Type:    "UDP-TUN",
			Network: "UDP",
			Rule:    string(rule.MatchRuler.MatcherResult().RuleType),
			Proxy:   proxy,
		})
		defer statistic.DefaultManager.Remove(udpTracker)
		conn = udpTracker
	}
	selector.ProxySelector.SelectPacket(proxy).Invoke(conn, info.Dst())
}

// isNeedHijackDNS check whether the dns request should be hijacked
// Only available for DNS over UDP and TCP with 53 port
func (handler *enhancerHandler) isNeedHijackDNS(addr netip.AddrPort) bool {
	// Over system config dns
	if addr.Addr().IsLoopback() && addr.Port() == 53 {
		return true
	}
	// Over fake dns ip
	if addr.Port() == uint16(handler.owner.fakeDns.Port) && addr.Addr().Compare(handler.owner.nameserver) == 0 {
		return true
	}
	// Over others dns ip
	for _, dns := range handler.owner.config.DnsHijack {
		// xxxxx:53 || any:53
		if addr.Compare(dns) == 0 || (dns.Addr().IsUnspecified() && addr.Port() == 53) {
			return true
		}
	}
	return false
}

// Refer to https://datatracker.ietf.org/doc/html/rfc7766#section-8
func (handler *enhancerHandler) hijackDNSForTCP(conn net.Conn) error {
	logger.Logger.Infof("hijack DNS over TCP request from %s", conn.LocalAddr())

	b := handler.pool.Get()
	defer handler.pool.Put(b)

	conn.SetReadDeadline(time.Now().Add(dnsMsgTimeout))
	n, err := io.ReadFull(conn, (*b)[:2])
	if err != nil {
		return err
	}
	if n < 2 {
		return dns.ErrShortRead
	}

	n = int((*b)[0])<<8 | int((*b)[1])
	conn.SetReadDeadline(time.Now().Add(dnsMsgTimeout))
	if n, err = io.ReadFull(conn, (*b)[:n]); err != nil {
		return err
	}
	req := dns.Msg{}
	if err = req.Unpack((*b)[:n]); err != nil {
		return err
	}
	// Response a dns reply with fake ip to client over TCP
	reply, err := resolver.DefaultResolver.Query(&req)
	if err != nil {
		return err
	}
	data, err := reply.PackBuffer(*b)
	if err != nil {
		return err
	}
	n = len(data)
	if _, err = conn.Write([]byte{byte(n >> 8), byte(n)}); err != nil {
		return err
	}
	conn.Write(data)
	return nil
}

func (handler *enhancerHandler) hijackDNSForUDP(conn net.PacketConn) error {
	logger.Logger.Infof("hijack DNS over UDP request from %s", conn.LocalAddr())

	b := handler.pool.Get()
	defer handler.pool.Put(b)

	conn.SetReadDeadline(time.Now().Add(dnsMsgTimeout))
	n, srcAddr, err := conn.ReadFrom(*b)
	if err != nil {
		return err
	}
	req := dns.Msg{}
	if err = req.Unpack((*b)[:n]); err != nil {
		return err
	}

	// Response a dns reply with fake ip to client over UDP
	reply, err := resolver.DefaultResolver.Query(&req)
	if err != nil {
		return err
	}
	data, err := reply.PackBuffer(*b)
	if err != nil {
		return err
	}
	conn.WriteTo(data, srcAddr)
	return nil
}
