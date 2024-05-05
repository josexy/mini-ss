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
	"net/url"
	"time"

	"github.com/josexy/mini-ss/bufferpool"
	"github.com/miekg/dns"
)

const dohMimeType = "application/dns-message"

type DnsClient struct {
	method string
	addr   string
	dnsC   *dns.Client
	httpC  *http.Client
	pool   *bufferpool.BufferPool
	// TODO: avoiding dns query endless loop for tun mode
	isFallback    bool
	fallbackAddrs []string
}

func NewDnsClient(dnsNet string, addr string, defaultDnsTimeout time.Duration) *DnsClient {
	client := &DnsClient{
		addr: addr,
		pool: bufferpool.NewBufferPool(4096 * 2),
	}

	if dnsNet == "https" {
		client.method = http.MethodGet
		client.httpC = &http.Client{
			Timeout: defaultDnsTimeout,
			Transport: &http.Transport{
				TLSHandshakeTimeout: time.Second * 5,
			},
		}
	} else {
		client.dnsC = &dns.Client{
			Net:          dnsNet,
			Timeout:      defaultDnsTimeout,
			ReadTimeout:  defaultDnsTimeout,
			WriteTimeout: defaultDnsTimeout,
		}
		if dnsNet == "tcp-tls" {
			host, _, _ := net.SplitHostPort(addr)
			client.dnsC.TLSConfig = &tls.Config{
				ServerName: host,
			}
		}
	}
	return client
}

func (c *DnsClient) ExchangeContext(ctx context.Context, request *dns.Msg) (reply *dns.Msg, err error) {
	if c.dnsC != nil {
		reply, _, err = c.dnsC.ExchangeContext(ctx, request, c.addr)
	} else {
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

	replyMsgData, err := io.ReadAll(httpRsp.Body)
	if err != nil {
		return
	}

	reply = new(dns.Msg)
	reply.Unpack(replyMsgData)
	return
}
