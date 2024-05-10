package transport

import (
	"context"
	"errors"
	"net"

	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/netstackgo/bind"
)

type tcpDialer struct{}

// FIXME: need dual stack dial?
func (d *tcpDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	if options.DefaultOptions.OutboundInterface == "" {
		dialer := &net.Dialer{Timeout: DefaultDialTimeout}
		return dialer.DialContext(ctx, "tcp", addr)
	}
	network := "tcp"
	switch network {
	case "tcp4", "tcp6":
		return d.dialSingle(ctx, network, addr)
	case "tcp":
		return d.dualStackDialContext(ctx, network, addr)
	default:
		return nil, errors.New("network invalid")
	}
}

func (d *tcpDialer) dialSingle(ctx context.Context, network string, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: DefaultDialTimeout}

	tcpAddr, err := resolver.DefaultResolver.ResolveTCPAddr(ctx, addr)
	if err != nil {
		return nil, err
	}
	if err := bind.BindToDeviceForTCP(options.DefaultOptions.OutboundInterface, dialer, network, tcpAddr.AddrPort().Addr()); err != nil {
		return nil, err
	}
	return dialer.DialContext(ctx, network, tcpAddr.String())
}

func (d *tcpDialer) dualStackDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	returned := make(chan struct{})
	defer close(returned)

	type dialResult struct {
		net.Conn
		error
		ipv6 bool
		done bool
	}
	results := make(chan dialResult)
	var primary, fallback dialResult

	startRacer := func(ctx context.Context, network, host string, ipv6 bool) {
		result := dialResult{ipv6: ipv6, done: true}
		defer func() {
			select {
			case results <- result:
			case <-returned:
				if result.Conn != nil {
					result.Conn.Close()
				}
			}
		}()
		result.Conn, result.error = d.dialSingle(ctx, network, addr)
	}

	go startRacer(ctx, network+"4", host, false)
	go startRacer(ctx, network+"6", host, true)

	for res := range results {
		if res.error == nil {
			return res.Conn, nil
		}

		if !res.ipv6 {
			primary = res
		} else {
			fallback = res
		}

		if primary.done && fallback.done {
			return nil, primary.error
		}
	}

	return nil, errors.New("never touched")
}
