package main

import (
	"context"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/config"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/statistic"
)

var logger = logx.NewDevelopment(
	logx.WithColor(true),
	logx.WithCaller(true, true, false, true),
	logx.WithJsonEncoder(),
	logx.WithLevel(true, true),
	logx.WithTime(true, func(t time.Time) string { return t.Format(time.TimeOnly) }),
)

type Config struct {
	Path  string         `json:"path"`
	Value *config.Config `json:"value"`
}

type App struct {
	ctx               context.Context
	running           bool
	server            *ss.ShadowsocksClient
	curCfg            *Config
	trafficSpeedCh    chan struct{}
	trafficSnapshotCh chan struct{}
}

func NewApp() *App {
	statistic.EnableStatistic = true
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	// stop server when closing window
	a.StopServer()
	return false
}
