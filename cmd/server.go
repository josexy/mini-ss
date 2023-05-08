package cmd

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "ss-server subcommand options",
	Example: "  mini-ss server -s :8388 -m aes-128-cfb -p 123456 -CV",
	Run: func(cmd *cobra.Command, args []string) {
		if err := StartServer(); err != nil {
			cmd.Help()
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&cfg.Server[0].Addr, "server", "s", "", "server listening address")
}

func StartServer() error {
	if len(cfg.Server) == 0 || cfg.Server[0].Addr == "" {
		return errors.New("server node is empty")
	}
	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(error); ok {
				logger.Logger.FatalBy(e)
			}
		}
	}()
	if err := startServer(); err != nil {
		logger.Logger.FatalBy(err)
	}
	return nil
}

func startServer() error {
	opts := cfg.BuildServerOptions()

	srv := ss.NewShadowsocksServer(opts...)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := srv.Start(); err != nil {
		return err
	}

	<-interrupt

	srv.Close()
	return nil
}
