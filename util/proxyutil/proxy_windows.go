package proxyutil

import (
	"net/url"

	"golang.org/x/sys/windows/registry"
)

const keyPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// SetSystemProxy for Windows, use http proxy to rewrite registry table
func SetSystemProxy(http, _ *url.URL) (err error) {
	if http == nil {
		return
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	key.SetDWordValue("ProxyEnable", 1)
	key.SetStringValue("ProxyServer", http.Host)
	return nil
}

func UnsetSystemProxy() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	key.SetDWordValue("ProxyEnable", 0)
	key.SetStringValue("ProxyServer", "")
	return nil
}
