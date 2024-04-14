package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fatih/color"
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
			MITM:    &config.MITMOption{},
			Tun:     &config.TunOption{},
			FakeDNS: &config.FakeDnsOption{},
		},
		Log: &config.LogConfig{},
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
	rootCmd.PersistentFlags().BoolVarP(&cfg.Log.Color, "color", "C", false, "enable output color mode")
	rootCmd.PersistentFlags().StringVarP(&cfg.Log.LogLevel, "level", "L", "info", "log level (trace, debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().IntVarP(&cfg.Log.VerboseLevel, "verbose-level", "V", 1, "verbose output level (0, 1, 2, 3)")
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
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.TLS.Mode, "ws-tls-mode", "", "ws tls mode(wss://) (tls, mtls)")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.TLS.KeyPath, "ws-tls-key", "", "ws tls key path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.TLS.CertPath, "ws-tls-cert", "", "ws tls cert path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.TLS.CAPath, "ws-tls-ca", "", "ws tls ca path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Ws.TLS.Hostname, "ws-tls-host", "", "ws tls common name")
	// obfs options
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Obfs.Host, "obfs-host", "www.baidu.com", "obfs host")
	// quic options
	rootCmd.PersistentFlags().IntVar(&cfg.Server[0].Quic.Conns, "quic-max-conn", 3, "maximum number of quic connections")
	// grpc options
	rootCmd.PersistentFlags().IntVar(&cfg.Server[0].Grpc.SendBufferSize, "grpc-send-buf", 0, "grpc send buffer size")
	rootCmd.PersistentFlags().IntVar(&cfg.Server[0].Grpc.RecvBufferSize, "grpc-recv-buf", 0, "grpc recv buffer size")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.TLS.Mode, "grpc-tls-mode", "", "grpc tls mode (tls, mtls)")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.TLS.KeyPath, "grpc-tls-key", "", "grpc tls key path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.TLS.CertPath, "grpc-tls-cert", "", "grpc tls cert path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.TLS.CAPath, "grpc-tls-ca", "", "grpc tls ca path")
	rootCmd.PersistentFlags().StringVar(&cfg.Server[0].Grpc.TLS.Hostname, "grpc-tls-host", "", "grpc tls common name")
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

	// disable logger
	if cfg.Log == nil || cfg.Log.VerboseLevel == 0 {
		logger.Logger = logx.NewLogContext().BuildConsoleLogger(logx.LevelTrace)
		return
	}

	var writer io.Writer = os.Stdout
	if cfg.Log.Color {
		writer = color.Output
	}
	logCtx := logx.NewLogContext().
		WithColor(cfg.Log.Color).
		WithTime(true, func(t time.Time) any { return t.Format(time.DateTime) }).
		WithCaller(true, true, true, true).
		WithLevel(true, true).
		WithEncoder(logx.Json).
		WithEscapeQuote(true).
		WithWriter(writer)

	switch cfg.Log.VerboseLevel {
	case 1:
		logCtx.WithCaller(false, false, false, false).WithTime(true, func(t time.Time) any { return t.Format(time.TimeOnly) })
	case 2:
		logCtx.WithCaller(true, true, false, true).WithTime(true, func(t time.Time) any { return t.Format(time.DateTime) })
	}
	var logLevel logx.LevelType
	switch cfg.Log.LogLevel {
	case "trace":
		logLevel = logx.LevelTrace
	case "debug":
		logLevel = logx.LevelDebug
	case "warn":
		logLevel = logx.LevelWarn
	case "error":
		logLevel = logx.LevelError
	case "fatal":
		logLevel = logx.LevelFatal
	case "panic":
		logLevel = logx.LevelPanic
	default:
		logLevel = logx.LevelInfo
	}
	logger.Logger = logCtx.BuildConsoleLogger(logLevel)
}
