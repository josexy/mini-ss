package server

import (
	"context"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/util/logger"
	"golang.org/x/sync/errgroup"
)

type ServerGroup struct {
	ctx        context.Context
	errg       *errgroup.Group
	serverList []Server
}

func NewServerGroup() *ServerGroup {
	errg, ctx := errgroup.WithContext(context.Background())
	return &ServerGroup{
		ctx:  ctx,
		errg: errg,
	}
}

func (g *ServerGroup) AddServer(server Server) {
	g.serverList = append(g.serverList, server)
}

func (g *ServerGroup) Start() error {
	for _, server := range g.serverList {
		srv := server
		g.errg.Go(func() error {
			logger.Logger.Info("start server", logx.String("type", srv.Type().String()), logx.String("listen", srv.LocalAddr()))
			return srv.Start(g.ctx)
		})
	}
	return g.errg.Wait()
}

func (g *ServerGroup) Close() error {
	var err error
	for _, server := range g.serverList {
		err = server.Close()
	}
	return err
}

func (g *ServerGroup) Len() int {
	return len(g.serverList)
}
