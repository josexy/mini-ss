package cmd

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "ss-server subcommand options",
	Example: "  mini-ss server -s :8388 -m aes-128-cfb -p 123456 -CV3",
	Run: func(cmd *cobra.Command, args []string) {
		if (len(cfg.Server) == 0 || cfg.Server[0].Addr == "") && configFile == "" {
			cmd.Help()
			return
		}
		StartServer()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&cfg.Server[0].Addr, "server", "s", "", "server listening address")
}

func StartServer() {
	if len(cfg.Server) == 0 || cfg.Server[0].Addr == "" {
		logger.Logger.FatalBy(errors.New("server node is empty"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(error); ok {
				logger.Logger.FatalBy(e)
			}
		}
	}()
	startServer()
}

func startServer() {
	logger.Logger.Info("build info", logx.String("version", Version), logx.String("git_commit", GitCommit))
	opts := cfg.BuildServerOptions()

	srv := ss.NewShadowsocksServer(opts...)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Logger.FatalBy(err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-interrupt

	srv.Close()
	time.Sleep(time.Millisecond * 300)
}
