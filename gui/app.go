package main

import (
	"context"
	"time"

	"github.com/fatih/color"
	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/config"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/statistic"
)

var logger = logx.NewLogContext().
	WithColor(true).
	WithCaller(true, true, false, false).
	WithLevel(true, true).
	WithEncoder(logx.Json).
	WithTime(true, func(t time.Time) any { return t.Format(time.TimeOnly) }).
	WithWriter(color.Output).BuildConsoleLogger(logx.LevelTrace)

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
