package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/config"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/ping"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/statistic"
	"github.com/josexy/mini-ss/util"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/proxyutil"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	sndSnapshotTime = 5 * time.Second
	sndTrafficTime  = time.Second
)

type App struct {
	ctx              context.Context
	cfg              *config.JsonConfig
	srv              *ss.ShadowsocksClient
	st               *ping.SpeedTest
	running          bool
	cfgContent       string
	cfgFilePath      string
	chTrafficTicker  chan struct{}
	chSnapshotTicker chan struct{}
}

func NewApp() *App {
	// disable logger
	logx.DisableColor = true
	logx.SetOutput(io.Discard)

	geoip.Data, _ = geoipdb.ReadFile("build/Country.mmdb")

	return &App{
		cfg: new(config.JsonConfig),
		st:  ping.NewSpeedTest(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) shutdown(ctx context.Context) {
	a.StopServer()
}

func (b *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

func (b *App) sendTrafficSpeed() {
	ticker := time.NewTicker(sndTrafficTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			download, upload := statistic.DefaultManager.TrafficSpeedTick()
			runtime.EventsEmit(b.ctx, "mini-ss-connection-traffic", download, upload)
		case <-b.chTrafficTicker:
			return
		}
	}
}

func (b *App) sendConnectionsSnapshot() {
	ticker := time.NewTicker(sndSnapshotTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			snapshot := statistic.DefaultManager.Snapshot()
			runtime.EventsEmit(b.ctx, "mini-ss-connection-snapshot", snapshot)
		case <-b.chSnapshotTicker:
			return
		}
	}
}

func (a *App) StartServer() error {
	if a.running {
		return errors.New("local server is running")
	}

	if len(a.cfg.Server) == 0 {
		return errors.New("server proxy not found")
	}

	if a.cfg.Rules == nil {
		a.cfg.Rules = &config.Rules{Mode: "global"}
	}
	a.srv = ss.NewShadowsocksClient(a.cfg.BuildSSLocalOptions()...)

	if err := a.srv.Start(); err != nil {
		return err
	}

	a.chTrafficTicker = make(chan struct{})
	a.chSnapshotTicker = make(chan struct{})
	go a.sendTrafficSpeed()
	go a.sendConnectionsSnapshot()
	a.running = true
	return nil
}

func (a *App) StopServer() error {
	if !a.running {
		return errors.New("local server was closed")
	}

	if a.cfg.Local.EnableTun {
		dnsutil.UnsetLocalDnsServer()
	}

	if a.cfg.Local.SystemProxy {
		proxyutil.UnsetSystemProxy()
	}

	a.chTrafficTicker <- struct{}{}
	a.chSnapshotTicker <- struct{}{}

	a.running = false
	return a.srv.Close()
}

func (a *App) GetSupportCipherMethods() []string {
	return supportCipherMethods
}

func (a *App) GetSupportTransportTypes() []string {
	return supportTransportTypes
}

func (a *App) GetSupportKcpCrypts() []string {
	return supportKcpCrypts
}

func (a *App) GetSupportKcpModes() []string {
	return supportKcpModes
}

func (a *App) GetJsonConfig() *config.JsonConfig {
	return a.cfg
}

func (a *App) GetJsonConfigFilePath() string {
	return a.cfgFilePath
}

func (a *App) GetJsonConfigContent() string {
	return a.cfgContent
}

func (a *App) GetAllInterfaceName() []string {
	return util.ResolveAllInterfaceName()
}

func (a *App) UpdateServerConfig(serverCfg *config.ServerJsonConfig) {
	index := -1
	for i, c := range a.cfg.Server {
		if c.Name == serverCfg.Name {
			index = i
			break
		}
	}
	if index == -1 {
		return
	}
	a.cfg.Server[index] = serverCfg
}

func (a *App) AddServerConfig(serverCfg *config.ServerJsonConfig) {
	a.cfg.Server = append(a.cfg.Server, serverCfg)
}

func (a *App) DeleteServerConfig(name string) {
	a.cfg.DeleteServerConfig(name)
}

func (a *App) SaveLocalConfig(localCfg *config.LocalJsonConfig, ifaceName string, authDetectIface bool) {
	a.cfg.Local = localCfg
	a.cfg.Iface = ifaceName
	a.cfg.AutoDetectIface = authDetectIface
}

func (a *App) LoadConfig() (*config.JsonConfig, error) {
	configFile, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "select a config json file",
	})
	if err != nil {
		return nil, err
	}

	return a.LoadConfigFile(configFile)
}

func (a *App) LoadConfigFile(path string) (*config.JsonConfig, error) {
	if err := a.loadConfig(path); err != nil {
		return nil, err
	}
	a.cfgFilePath = path
	return a.cfg, nil
}

func (a *App) loadConfig(path string) error {
	cfg, content, err := config.ParseJsonConfigFile(path)
	if err != nil {
		cfg = new(config.JsonConfig)
	}
	a.cfg, a.cfgContent = cfg, content
	return nil
}

func (a *App) Ping(addr string) (string, error) {
	rtt, err := ping.Ping(addr, 3)
	if err != nil {
		return "", err
	}
	return rtt.String(), nil
}

func (a *App) TcpPing(addr string) (string, error) {
	rtt, err := ping.TCPing(addr, 3)
	if err != nil {
		return "", err
	}
	return rtt.String(), nil
}

func (a *App) SpeedTest() (string, error) {
	var mode, socks, http, auth string
	if a.cfg.Local.SocksAddr != "" {
		// socks
		mode = "socks"
		socks = a.cfg.Local.SocksAddr
		auth = a.cfg.Local.SocksAuth
	} else if a.cfg.Local.HTTPAddr != "" {
		// http
		mode = "http"
		http = a.cfg.Local.HTTPAddr
		auth = a.cfg.Local.HTTPAuth
	} else {
		// tun
		mode = "tun"
	}

	a.st.SetTestProxy(mode, http, socks)
	a.st.SetAuth(auth)

	if err := a.st.Start(time.Minute); err != nil {
		return "", err
	}
	_, rate := a.st.Status()
	return rate, nil
}

func (a *App) StopSpeedTest() {
	a.st.Stop()
}

func (a *App) ExportCurrentConfig() error {
	configFile, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "save config file",
		DefaultFilename: "save-config.json",
	})
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(a.cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0660)
}
