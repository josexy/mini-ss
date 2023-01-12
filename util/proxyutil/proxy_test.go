package proxyutil

import (
	"net/url"
	"testing"
)

func TestSetSystemProxy(t *testing.T) {
	SetSystemProxy(&url.URL{
		Scheme: "http",
		User:   url.UserPassword("123", "456"),
		Host:   "127.0.0.1:10086",
	}, &url.URL{
		Scheme: "socks",
		Host:   "127.0.0.1:10087",
	})
}

func TestUnsetSystemProxy(t *testing.T) {
	UnsetSystemProxy()
}
