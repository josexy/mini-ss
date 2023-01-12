package proxyutil

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var deVars = []string{
	"XDG_CURRENT_DESKTOP",
	"GDMSESSION",
	"DESKTOP_SESSION",
}

var deList = []string{
	"ubuntu",
	"kde",
}

func checkDE() string {
	for _, de := range deList {
		for _, name := range deVars {
			if value, ok := os.LookupEnv(name); ok {
				value = strings.ToLower(value)
				if strings.Contains(value, de) {
					return de
				}
			}
		}
	}
	return "unknown"
}

func SetSystemProxy(http, socks *url.URL) (err error) {
	de := checkDE()
	if de == "unknown" {
		return errors.New("current desktop environment don't support GUI system proxy setting")
	}
	if http == nil && socks == nil {
		return
	}
	var shell string
	switch de {
	case "ubuntu":
		if http != nil {
			shell = `
gsettings set org.gnome.system.proxy mode "manual"
gsettings set org.gnome.system.proxy.http host "` + http.Hostname() + `"
gsettings set org.gnome.system.proxy.http port ` + http.Port() + `
gsettings set org.gnome.system.proxy.https host "` + http.Hostname() + `"
gsettings set org.gnome.system.proxy.https port ` + http.Port()
			if http.User != nil {
				password, _ := http.User.Password()
				shell += `
gsettings set org.gnome.system.proxy.http authentication-user "` + http.User.Username() + `"
gsettings set org.gnome.system.proxy.http authentication-password "` + password + `"`
			}
		}
		if socks != nil {
			shell += `
gsettings set org.gnome.system.proxy.socks host "` + socks.Hostname() + `"
gsettings set org.gnome.system.proxy.socks port ` + socks.Port()
		}
		shell += `
gsettings set org.gnome.system.proxy ignore-hosts "['localhost', '127.0.0.0/8', '::1', '192.168.0.0/16', '10.0.0.0/8', '172.16.0.0/12']"
`
	case "kde":
	}
	return exec.Command("sh", "-c", shell).Run()
}

func UnsetSystemProxy() error {
	de := checkDE()
	if de == "unknown" {
		return errors.New("current desktop environment don't support GUI system proxy setting")
	}
	var shell string
	switch de {
	case "ubuntu":
		shell = `
gsettings set org.gnome.system.proxy mode "none"
gsettings reset org.gnome.system.proxy.http host
gsettings reset org.gnome.system.proxy.http port
gsettings reset org.gnome.system.proxy.https host
gsettings reset org.gnome.system.proxy.https port 
gsettings reset org.gnome.system.proxy.http authentication-user
gsettings reset org.gnome.system.proxy.http authentication-password
gsettings reset org.gnome.system.proxy.socks host
gsettings reset org.gnome.system.proxy.socks port
gsettings reset org.gnome.system.proxy ignore-hosts
`
	case "kde":
	}
	return exec.Command("sh", "-c", shell).Run()
}
