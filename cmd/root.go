package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/config"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/spf13/cobra"
)

var Version = "unknown"

var rootCmd = &cobra.Command{
	Use:     "mini-ss",
	Short:   "mini shadowsocks server and client",
	Version: Version,
}

var (
	configFile string
	cfg        = &config.Config{
		Server: []*config.ServerConfig{{
			Kcp:  &config.KcpOption{},
			Ws:   &config.WsOption{},
			Quic: &config.QuicOption{},
			Obfs: &config.ObfsOption{},
			Grpc: &config.GrpcOption{},
			SSR:  &config.SSROption{},
		}},
		Local: &config.LocalConfig{
			Tun:     &config.TunOption{},
			FakeDNS: &config.FakeDnsOption{},
		},
		Rules: &config.Rules{
			Mode:  "global", // default global rule
			Match: &config.Match{},
		},
	}
)

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// shadowsocks options
	rootCmd.PersistentFlags().StringVarP(&cfg.Server[0].Method, "method", "m", "none", "the cipher method between ss-local and ss-server")
	rootCmd.PersistentFlags().StringVarP(&cfg.Server[0].Password, "password", "p", "", "the password for cipher method")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "server or client configuration file")
	rootCmd.PersistentFlags().StringVarP(&cfg.Server[0].Transport, "transport", "t", "default", "the transport type between ss-local and ss-server (default, kcp, quic, ws)")
	// logger options
	rootCmd.PersistentFlags().BoolVarP(&cfg.Color, "color", "C", false, "enable output color mode")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "V", false, "enable verbose mode")
	rootCmd.PersistentFlags().IntVarP(&cfg.VerboseLevel, "verbose-level", "L", 2, "verbose output level (1, 2, 3)")
	// kcp options
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Kcp.Crypt, "kcp-crypt", "none", "kcp encrypt and decrypt method")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Kcp.Key, "kcp-key", "", "kcp encrypt and decrypt key")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Kcp.Mode, "kcp-mode", "normal", "kcp parameters (normal, fast1, fast2, fast3)")
	rootCmd.PersistentFlags().BoolVar(&cfg.Server[0].Kcp.Compress, "kcp-compress", false, "enable kcp snappy algorithm")
	rootCmd.PersistentFlags().IntVar(&cfg.Server[0].Kcp.Conns, "kcp-mux-conn", 3, "maximum number of kcp connections")
	// websocket options
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.Host, "ws-host", "www.baidu.com", "websocket host")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.Path, "ws-path", "/ws", "websocket request path")
	rootCmd.PersistentFlags().BoolVar(&cfg.Server[0].Ws.Compress, "ws-compress", false, "enable data compression")
	rootCmd.PersistentFlags().BoolVar(&cfg.Server[0].Ws.TLS, "ws-tls", false, "enable secure transmission (wss://)")
	// obfs options
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Obfs.Host, "obfs-host", "www.baidu.com", "obfs host")
	// quic options
	rootCmd.PersistentFlags().IntVar(&cfg.Server[0].Quic.Conns, "quic-max-conn", 3, "maximum number of quic connections")
	// grpc options
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.Hostname, "grpc-host", "", "grpc hostname")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.KeyPath, "grpc-key-path", "", "grpc mTLS key path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.CertPath, "grpc-cert-path", "", "grpc mTLS cert path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.CAPath, "grpc-ca-path", "", "grpc mTLS CA path")
	rootCmd.PersistentFlags().BoolVar(&cfg.Server[0].Grpc.TLS, "grpc-tls", false, "enable grpc mTLS")
}

func initConfig() {
	// overwrite default options if use config file
	if configFile != "" {
		var err error
		cfg, err = config.ParseConfigFile(configFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if !cfg.Verbose {
		logger.Logger = logx.NewNop()
		return
	}

	opts := []logx.ConfigOption{
		logx.WithColor(cfg.Color),
		logx.WithLevel(true, true),
		logx.WithJsonEncoder(),
		logx.WithEscapeQuote(true),
	}

	switch cfg.VerboseLevel {
	case 1:
		break
	case 3:
		opts = append(opts,
			logx.WithCaller(true, true, true, true),
			logx.WithTime(true, func(t time.Time) string { return t.Format(time.DateTime) }),
		)
	default:
		opts = append(opts,
			logx.WithCaller(true, true, false, true),
			logx.WithTime(true, func(t time.Time) string { return t.Format(time.TimeOnly) }),
		)
	}
	logger.Logger = logx.NewDevelopment(opts...)
}
