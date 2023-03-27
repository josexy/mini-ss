package transport

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/josexy/netstackgo/bind"
)

type tcpDialer struct{}

func (d *tcpDialer) Dial(addr string) (net.Conn, error) {
	if DefaultDialerOutboundOption.Interface == "" {
		dialer := &net.Dialer{Timeout: dialTimeout}
		return dialer.DialContext(context.Background(), "tcp", addr)
	}
	network := "tcp"
	switch network {
	case "tcp4", "tcp6":
		return dialSingle(context.Background(), network, addr)
	case "tcp":
		return dualStackDialContext(context.Background(), network, addr)
	default:
		return nil, errors.New("network invalid")
	}
}

func dialSingle(ctx context.Context, network string, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: dialTimeout}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ip := resolveIP(host)

	if !ip.IsValid() {
		return nil, fmt.Errorf("can not look up host: %s", addr)
	}

	if err := bind.BindToDeviceForTCP(DefaultDialerOutboundOption.Interface, dialer, network, ip); err != nil {
		return nil, err
	}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
}

func dualStackDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
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
		result.Conn, result.error = dialSingle(ctx, network, addr)
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
