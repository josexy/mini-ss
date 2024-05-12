package resolver

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/netstackgo/bind"
	"github.com/miekg/dns"
)

const dohMimeType = "application/dns-message"

type DnsClient struct {
	method string
	host   string
	addr   string
	dnsC   *dns.Client
	httpC  *http.Client
	pool   *bufferpool.BufferPool
}

func NewDnsClient(dnsNet string, addr string, defaultDnsTimeout time.Duration) *DnsClient {
	client := &DnsClient{
		addr: addr,
		pool: bufferpool.NewBufferPool(4096 * 2),
	}

	if dnsNet == "https" {
		urlres, _ := url.Parse(addr)
		client.host = urlres.Hostname()
		client.method = http.MethodGet
		client.httpC = &http.Client{
			Timeout: defaultDnsTimeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					dialer := &net.Dialer{Timeout: defaultDnsTimeout}
					if options.DefaultOptions.OutboundInterface != "" {
						ip, _ := netip.ParseAddr(client.host)
						bind.BindToDeviceForTCP(options.DefaultOptions.OutboundInterface, dialer, "tcp", ip)
					}
					return dialer.DialContext(ctx, network, addr)
				},
				TLSHandshakeTimeout: time.Second * 5,
				IdleConnTimeout:     time.Second * 5,
			},
		}
	} else {
		client.host, _, _ = net.SplitHostPort(addr)

		dialer := &net.Dialer{Timeout: defaultDnsTimeout}
		if options.DefaultOptions.OutboundInterface != "" {
			ip, _ := netip.ParseAddr(client.host)
			bind.BindToDeviceForTCP(options.DefaultOptions.OutboundInterface, dialer, "tcp", ip)
		}

		client.dnsC = &dns.Client{
			Net:          dnsNet,
			Timeout:      defaultDnsTimeout,
			ReadTimeout:  defaultDnsTimeout,
			WriteTimeout: defaultDnsTimeout,
			Dialer:       dialer,
		}
		if dnsNet == "tcp-tls" {
			client.dnsC.TLSConfig = &tls.Config{
				ServerName: client.host,
			}
		}
	}
	return client
}

func (c *DnsClient) ExchangeContext(ctx context.Context, request *dns.Msg) (reply *dns.Msg, err error) {
	defer func() {
		if err != nil {
			logger.Logger.Errorf("dns exchange failed: %s, %s, %s", err.Error(), request.Question[0].String(), c.addr)
		}
	}()
	domain := dnsutil.TrimDomain(request.Question[0].Name)
	if c.dnsC != nil {
		logger.Logger.Tracef("dns exchange: %s for domain: %s", c.addr, domain)
		reply, _, err = c.dnsC.ExchangeContext(ctx, request, c.addr)
	} else {
		logger.Logger.Tracef("dns exchange: %s for domain: %s", c.addr, domain)
		reply, err = c.exchangeDoH(ctx, request)
	}
	return
}

func (c *DnsClient) exchangeDoH(ctx context.Context, request *dns.Msg) (reply *dns.Msg, err error) {
	buf := c.pool.Get()
	defer c.pool.Put(buf)

	reqMsgData, err := request.PackBuffer(*buf)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if c.method == http.MethodPost {
		bodyReader = bytes.NewReader(reqMsgData)
	}

	httpReq, err := http.NewRequestWithContext(ctx, c.method, c.addr, bodyReader)
	if err != nil {
		return
	}

	httpReq.Header.Set("Accept", dohMimeType)
	if c.method == http.MethodPost {
		httpReq.Header.Set("Content-Type", dohMimeType)
	} else {
		values := make(url.Values, 1)
		values.Set("dns", base64.RawURLEncoding.EncodeToString(reqMsgData))
		httpReq.URL.RawQuery = values.Encode()
	}

	httpRsp, err := c.httpC.Do(httpReq)
	if err != nil {
		return
	}
	defer httpRsp.Body.Close()

	if httpRsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dns: server returned HTTP %d error: %s", httpRsp.StatusCode, httpRsp.Status)
	}

	if ct := httpRsp.Header.Get("Content-Type"); ct != dohMimeType {
		return nil, fmt.Errorf("dns: unexpected Content-Type %s; expected %s", ct, dohMimeType)
	}

	readBody := func(r io.Reader, b []byte) ([]byte, error) {
		var i int
		for {
			n, err := r.Read(b[i:])
			i += n
			if err != nil {
				if err == io.EOF {
					err = nil
				}
				return b[:i], err
			}
		}
	}
	replyMsgData, err := readBody(httpRsp.Body, *buf)
	if err != nil {
		return
	}

	reply = new(dns.Msg)
	reply.Unpack(replyMsgData)
	return
}
