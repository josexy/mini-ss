package proxyutil

import (
	"net/url"
	"os/exec"
)

// SetSystemProxy set http/https and socks proxy
// httpProxy: http://user:pass@127.0.0.1:8888
// httpProxy: socks://user:pass@127.0.0.1:9999
func SetSystemProxy(http, socks *url.URL) (err error) {
	if http == nil && socks == nil {
		return
	}
	shell := `
function scutil_query {
key=$1
scutil <<EOT
open
get $key
d.show
close
EOT
}
SERVICE_GUID=$(scutil_query State:/Network/Global/IPv4 | grep "PrimaryService" | awk '{print $3}')
currentservice=$(scutil_query Setup:/Network/Service/$SERVICE_GUID | grep "UserDefinedName" | awk -F': ' '{print $2}')

networksetup -setwebproxystate "$currentservice" on
networksetup -setsecurewebproxystate "$currentservice" on
networksetup -setsocksfirewallproxystate "$currentservice" on
`
	// http proxy
	if http != nil {
		if http.User != nil {
			password, _ := http.User.Password()
			shell += `networksetup -setwebproxy "$currentservice" ` + http.Hostname() + " " + http.Port() + " on " + http.User.Username() + " " + password
			shell += "\n"
			shell += `networksetup -setsecurewebproxy "$currentservice" ` + http.Hostname() + " " + http.Port() + " on " + http.User.Username() + " " + password
		} else {
			shell += `networksetup -setwebproxy "$currentservice" ` + http.Hostname() + " " + http.Port()
			shell += "\n"
			shell += `networksetup -setsecurewebproxy "$currentservice" ` + http.Hostname() + " " + http.Port()
		}
	}
	if socks != nil {
		shell += "\n"
		// socks proxy
		if socks.User != nil {
			password, _ := socks.User.Password()
			shell += `networksetup -setsocksfirewallproxy "$currentservice" ` + socks.Hostname() + " " + socks.Port() + " on " + socks.User.Username() + " " + password
		} else {
			shell += `networksetup -setsocksfirewallproxy "$currentservice" ` + socks.Hostname() + " " + socks.Port()
		}
	}
	// bypass domain
	shell += `
networksetup -setproxybypassdomains "$currentservice" 192.168.0.0/16 10.0.0.0/8 172.16.0.0/12 127.0.0.1 localhost "*.local" timestamp.apple.com
`
	return exec.Command("bash", "-c", shell).Run()
}

// UnsetSystemProxy clear and disable system proxy for http/https and socks proxy
func UnsetSystemProxy() error {
	shell := `
function scutil_query {
key=$1
scutil <<EOT
open
get $key
d.show
close
EOT
}
SERVICE_GUID=$(scutil_query State:/Network/Global/IPv4 | grep "PrimaryService" | awk '{print $3}')
currentservice=$(scutil_query Setup:/Network/Service/$SERVICE_GUID | grep "UserDefinedName" | awk -F': ' '{print $2}')

networksetup -setwebproxy "$currentservice" "" "" off "" ""
networksetup -setsecurewebproxy "$currentservice" "" "" off "" ""
networksetup -setsocksfirewallproxy "$currentservice" "" "" off "" ""

networksetup -setwebproxystate "$currentservice" off
networksetup -setsecurewebproxystate "$currentservice" off
networksetup -setsocksfirewallproxystate "$currentservice" off

networksetup -setproxybypassdomains "$currentservice" Empty
`
	return exec.Command("bash", "-c", shell).Run()
}
