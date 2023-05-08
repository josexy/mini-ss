package main

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/config"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/rule"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/util/dnsutil"
	loggerPkg "github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/mini-ss/util/ping"
	"github.com/josexy/proxyutil"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/yaml.v2"
)

func (a *App) LoadConfig() (*Config, error) {
	configPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "select a config yaml file",
	})
	if err != nil {
		logger.ErrorBy(err)
		return nil, err
	}
	cfg, err := config.ParseConfigFile(configPath)
	if err != nil {
		logger.ErrorBy(err)
		return nil, err
	}
	logger.Info("load config file", logx.String("path", configPath))
	a.curCfg = &Config{
		Path:  configPath,
		Value: cfg,
	}
	return a.curCfg, nil
}

func (a *App) SaveConfig(cfg *config.Config) error {
	configPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "select a config yaml file",
		DefaultFilename: "output.yaml",
	})
	if err != nil {
		logger.ErrorBy(err)
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		logger.ErrorBy(err)
		return err
	}
	logger.Info("save config file", logx.String("path", configPath))
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		logger.ErrorBy(err)
	}
	return err
}

func (a *App) ListOutboundInterfaces() ([]string, error) {
	var list []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		hasIPv4Addr := false
		for _, addr := range addrs {
			ipNet := netip.MustParsePrefix(addr.String())
			if ipNet.Addr().Is4() {
				hasIPv4Addr = true
				break
			}
		}
		if hasIPv4Addr {
			list = append(list, iface.Name)
		}
	}
	return list, nil
}

func (a *App) StartServer(cfg *config.Config) error {
	if a.running {
		return errors.New("server has already run")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := geoip.OpenDB(filepath.Join(home, ".config/clash/Country.mmdb")); err != nil {
		return err
	}

	loggerPkg.Logger = logger

	logger.Info("start server")
	a.server = ss.NewShadowsocksClient(cfg.BuildSSLocalOptions()...)
	err = a.server.Start()
	if err == nil {
		a.running = true

		a.trafficSpeedCh = make(chan struct{})
		a.trafficSnapshotCh = make(chan struct{})
		go a.startTrafficSpeed()
		go a.startTrafficSnapshot()
	}
	a.curCfg.Value = cfg
	return err
}

func (a *App) StopServer() error {
	if a.server == nil || !a.running {
		return errors.New("server not run")
	}
	a.trafficSpeedCh <- struct{}{}
	a.trafficSnapshotCh <- struct{}{}
	statistic.DefaultManager.Reset()

	logger.Info("stop server")
	a.running = false
	if a.curCfg.Value.Local.EnableTun {
		dnsutil.UnsetLocalDnsServer()
	}
	if a.curCfg.Value.Local.SystemProxy {
		proxyutil.UnsetSystemProxy()
	}
	return a.server.Close()
}

func (a *App) SpeedTest(addr string) (res string, err error) {
	logger.Info("speed test", logx.String("address", addr))

	var d time.Duration
	d, err = ping.TCPing(addr, 2)
	if err != nil && err != ping.ErrTimeout {
		return
	}
	if err == ping.ErrTimeout {
		res = err.Error()
		err = nil
	} else {
		res = d.String()
	}

	return
}

func (a *App) ChangeGlobalTo(proxyName string) {
	if rule.MatchRuler == nil {
		return
	}
	rule.MatchRuler.GlobalTo = proxyName
	logger.Info("change global proxy", logx.String("name", proxyName))
}

func (a *App) ChangeDirectTo(proxyName string) {
	if rule.MatchRuler == nil {
		return
	}
	rule.MatchRuler.DirectTo = proxyName
	logger.Info("change direct proxy", logx.String("name", proxyName))
}

func (a *App) ListTransports() []string { return supportTransportList }

func (a *App) ListMethods() []string { return supportMethodList }

func (a *App) ListKcpModes() []string { return supportKcpModeList }

func (a *App) ListKcpCrypts() []string { return supportKcpCryptList }

func (a *App) ListSSRObfs() []string { return supportSSRObfsList }

func (a *App) ListSSRProtocols() []string { return supportSSRProtocolList }

func (a *App) AddServerConfig(cfg *config.ServerConfig) error {
	a.curCfg.Value.Server = append(a.curCfg.Value.Server, cfg)
	logger.Info("add server config", logx.String("name", cfg.Name))
	return writeYaml(a.curCfg.Path, a.curCfg.Value)
}

func (a *App) UpdateServerConfig(cfg *config.ServerConfig) error {
	servers := a.curCfg.Value.Server
	for i := 0; i < len(servers); i++ {
		if servers[i].Name == cfg.Name {
			logger.Info("update server config", logx.String("name", cfg.Name))
			servers[i] = cfg
			return writeYaml(a.curCfg.Path, a.curCfg.Value)
		}
	}
	return nil
}

func (a *App) DeleteServerConfig(name string) error {
	index := -1
	servers := a.curCfg.Value.Server
	for i := 0; i < len(servers); i++ {
		if servers[i].Name == name {
			index = i
			break
		}
	}
	if index == -1 {
		return nil
	}
	logger.Info("delete server config", logx.String("name", name))
	a.curCfg.Value.Server = append(a.curCfg.Value.Server[:index], a.curCfg.Value.Server[index+1:]...)
	return writeYaml(a.curCfg.Path, a.curCfg.Value)
}

func (a *App) startTrafficSpeed() {
	ticker := time.NewTicker(statistic.TrafficSpeedTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			second := statistic.TrafficSpeedTime / time.Second
			download, upload := statistic.DefaultManager.TrafficSpeed()
			downloadStr := formatBytes(float64(download) / float64(second))
			uploadStr := formatBytes(float64(upload) / float64(second))
			runtime.EventsEmit(a.ctx, event_traffic_speed, downloadStr, uploadStr)
		case <-a.trafficSpeedCh:
			return
		}
	}
}

func (a *App) startTrafficSnapshot() {
	ticker := time.NewTicker(statistic.TrafficSnapshotTime)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snapshot := statistic.DefaultManager.DumpSnapshot()
			runtime.EventsEmit(a.ctx, event_traffic_snapshot, snapshot)
		case <-a.trafficSnapshotCh:
			return
		}
	}
}
