package enhancer

import (
	"net"
	"strconv"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/selector"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/netstackgo"
	"github.com/miekg/dns"
)

const DefaultMTU = 1350

var pool = bufferpool.NewBufferPool(constant.MaxUdpBufferSize)

type enhancerHandler struct{ owner *Enhancer }

func newEnhancerHandler(eh *Enhancer) *enhancerHandler { return &enhancerHandler{owner: eh} }

func (handler *enhancerHandler) relayFakeDnsRequest(conn net.PacketConn) error {
	b := pool.Get()
	defer pool.Put(b)

	n, srcAddr, err := conn.ReadFrom(*b)
	if err != nil {
		return err
	}
	req := dns.Msg{}
	if err = req.Unpack((*b)[:n]); err != nil {
		return err
	}

	// for tun mode, we can directly return the fake ip address corresponding to the domain name
	reply, err := resolver.DefaultResolver.Query(&req)
	if err != nil {
		return err
	}
	data, _ := reply.PackBuffer(*b)
	conn.WriteTo(data, srcAddr)
	return nil
}

func (handler *enhancerHandler) HandleTCPConn(info *netstackgo.ConnTuple, conn net.Conn) {
	// the target address(info.DstIP) may be a fake ip address or real ip address
	// for example `curl www.google.com` or `curl 74.125.24.103:80`

	// note: The following request methods will not handle the remote request address,
	// but will be handled by the proxy node server
	// `curl --socks5 127.0.0.1:10088 74.125.24.103:80`
	// `curl --proxy 127.0.0.1:10088 74.125.24.103:80`
	// `curl --socks5 127.0.0.1:10088 www.google.com`
	// `curl --proxy 127.0.0.1:10088 www.google.com`

	var remote string
	fakeDnsRecord := resolver.DefaultResolver.FindByIP(info.DstIP)
	if fakeDnsRecord == nil {
		remote = info.DstIP.String()
	} else {
		remote = fakeDnsRecord.Domain
	}

	if !rule.MatchRuler.Match(&remote) {
		return
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	remoteAddr := net.JoinHostPort(remote, strconv.Itoa(int(info.DstPort)))

	logger.Logger.Debug("tcp-tun",
		logx.String("src", info.Src()),
		logx.String("dst", info.Dst()),
		logx.String("remote", remoteAddr),
	)

	if err := selector.ProxySelector.Select(proxy)(conn, remoteAddr); err != nil {
		logger.Logger.ErrorBy(err)
	}
}

func (handler *enhancerHandler) HandleUDPConn(info *netstackgo.ConnTuple, conn net.PacketConn) {
	// relay dns packet
	if info.DstPort == uint16(handler.owner.fakeDns.Port) && info.DstIP.Compare(handler.owner.nameserver) == 0 {
		handler.relayFakeDnsRequest(conn)
		return
	}

	// discard udp fake ip
	if handler.owner.config.tunCidr.Contains(info.DstIP) {
		return
	}

	// for UDP request matching, support GeoIP and IP-CIDR
	remote := info.DstIP.String()
	if !rule.MatchRuler.Match(&remote) {
		return
	}

	proxy, err := rule.MatchRuler.Select()
	if err != nil {
		logger.Logger.ErrorBy(err)
		return
	}

	logger.Logger.Debug("udp-tun", logx.String("src", info.Src()), logx.String("dst", info.Dst()))

	selector.ProxySelector.SelectPacket(proxy)(conn, info.Dst())
}
