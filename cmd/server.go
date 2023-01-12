package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/ss"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:     "server",
	Short:   "ss-server subcommand options",
	Example: "  mini-ss server -s :8388 -m aes-128-cfb -p 123456 -CV",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(error); ok {
					logx.FatalBy(e)
				}
			}
		}()
		if len(jsonCfg.Server) == 0 {
			logx.Fatal("server node is empty")
		}
		if jsonCfg.Server[0].Addr == "" {
			cmd.Help()
			return
		}
		if err := startServer(); err != nil {
			logx.FatalBy(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().StringVarP(&jsonCfg.Server[0].Addr, "server", "s", "", "server listening address")
}

func startServer() error {
	opts := jsonCfg.BuildServerOptions()

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
