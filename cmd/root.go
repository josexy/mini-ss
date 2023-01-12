package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/config"
	"github.com/josexy/mini-ss/ping"
	"github.com/spf13/cobra"
)

var Version = "unknown"

var rootCmd = &cobra.Command{
	Use:     "mini-ss",
	Short:   "mini shadowsocks server and client",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		// speed test
		if speedT.mode != "" {
			var rtt time.Duration
			var rate string
			var err error
			if speedT.testUrl != "" {
				ping.SetSpeedTestUrl(speedT.testUrl)
			}
			switch speedT.mode {
			case "tun":
				rtt, rate, err = ping.TunSpeedTest(time.Minute)
			case "http":
				rtt, rate, err = ping.HttpSpeedTest(speedT.proxy, time.Minute)
			case "socks":
				rtt, rate, err = ping.SocksSpeedTest(speedT.proxy, time.Minute)
			default:
				cmd.Help()
				return
			}
			if err != nil {
				fmt.Printf("speed test err: %v\n", err)
			} else {
				fmt.Printf("speed test rtt: %v, rate: %s\n", rtt, rate)
			}
			return
		}

		// ping by config file
		if configFile != "" {
			// clear
			pingT.serverList = pingT.serverList[:0]
			for _, srvCfg := range jsonCfg.Server {
				pingT.serverList = append(pingT.serverList, srvCfg.Addr)
			}
		}

		// ping test
		var rtts []time.Duration
		var errs []error
		switch {
		case pingT.ping: // raw icmp ping
			rtts, errs = ping.PingList(pingT.serverList, pingT.count)
		case pingT.tcping: // tcp ping
			rtts, errs = ping.TCPingList(pingT.serverList, pingT.count)
		case pingT.httping: // http get ping
			rtts, errs = ping.HTTPingList(pingT.serverList, pingT.count)
		default:
			cmd.Help()
			return
		}
		for i := 0; i < len(rtts); i++ {
			if errs[i] == nil {
				fmt.Printf("ping to %q rtt: %s\n", pingT.serverList[i], rtts[i])
			} else {
				fmt.Printf("ping to %q err: %v\n", pingT.serverList[i], errs[i])
			}
		}
	},
}

type pingTest struct {
	ping       bool
	tcping     bool
	httping    bool
	serverList []string
	count      int
}

type speedTest struct {
	mode    string
	proxy   string
	testUrl string
}

var (
	configFile string
	jsonCfg    = &config.JsonConfig{
		Server: []*config.ServerJsonConfig{{
			Kcp:  &config.KcpOption{},
			Ws:   &config.WsOption{},
			Quic: &config.QuicOption{},
			Obfs: &config.ObfsOption{},
			SSR:  &config.SSROption{},
		}},
		Local: &config.LocalJsonConfig{
			Tun:     &config.TunOption{},
			FakeDNS: &config.FakeDnsOption{},
		},
		Rules: &config.Rules{
			Mode:  "global", // default global rule
			Match: &config.Match{},
		},
	}
	pingT  pingTest
	speedT speedTest
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
	rootCmd.PersistentFlags().StringVarP(&jsonCfg.Server[0].Method, "method", "m", "none", "the cipher method between ss-local and ss-server")
	rootCmd.PersistentFlags().StringVarP(&jsonCfg.Server[0].Password, "password", "p", "", "the password for cipher method")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "server or client configuration file")
	rootCmd.PersistentFlags().StringVarP(&jsonCfg.Server[0].Transport, "transport", "t", "default", "the transport type between ss-local and ss-server (default, kcp, quic, ws)")
	// logger options
	rootCmd.PersistentFlags().BoolVarP(&jsonCfg.Color, "color", "C", false, "enable output color mode")
	rootCmd.PersistentFlags().BoolVarP(&jsonCfg.Verbose, "verbose", "V", false, "enable verbose mode")
	rootCmd.PersistentFlags().IntVarP(&jsonCfg.VerboseLevel, "verbose-level", "L", 2, "verbose output level (1, 2, 3)")
	// kcp options
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Server[0].Kcp.Crypt, "kcp-crypt", "none", "kcp encrypt and decrypt method")
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Server[0].Kcp.Key, "kcp-key", "", "kcp encrypt and decrypt key")
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Server[0].Kcp.Mode, "kcp-mode", "normal", "kcp parameters (normal, fast1, fast2, fast3)")
	rootCmd.PersistentFlags().BoolVar(&jsonCfg.Server[0].Kcp.Compress, "kcp-compress", false, "enable kcp snappy algorithm")
	rootCmd.PersistentFlags().IntVar(&jsonCfg.Server[0].Kcp.Conns, "kcp-mux-conn", 3, "maximum number of kcp connections")
	// websocket options
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Server[0].Ws.Host, "ws-host", "www.baidu.com", "websocket host")
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Server[0].Ws.Path, "ws-path", "/ws", "websocket request path")
	rootCmd.PersistentFlags().BoolVar(&jsonCfg.Server[0].Ws.Compress, "ws-compress", false, "enable data compression")
	rootCmd.PersistentFlags().BoolVar(&jsonCfg.Server[0].Ws.TLS, "ws-tls", false, "enable secure transmission (wss://)")
	// obfs options
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Server[0].Obfs.Host, "obfs-host", "www.baidu.com", "obfs host")
	rootCmd.PersistentFlags().BoolVar(&jsonCfg.Server[0].Obfs.TLS, "obfs-tls", false, "enable secure transmission")
	// quic options
	rootCmd.PersistentFlags().IntVar(&jsonCfg.Server[0].Quic.Conns, "quic-max-conn", 3, "maximum number of quic connections")
	// interface
	rootCmd.PersistentFlags().StringVar(&jsonCfg.Iface, "iface", "", "bind outbound interface")
	rootCmd.PersistentFlags().BoolVar(&jsonCfg.AutoDetectIface, "auto-detect-iface", false, "enable auto-detect interface")

	// utility
	rootCmd.Flags().StringSliceVar(&pingT.serverList, "ping-list", nil, "ping test servers")
	rootCmd.Flags().BoolVar(&pingT.ping, "ping", false, "enable icmp ping test")
	rootCmd.Flags().BoolVar(&pingT.tcping, "tcp-ping", false, "enable tcp ping test")
	rootCmd.Flags().BoolVar(&pingT.httping, "http-ping", false, "enable http get method ping test")
	rootCmd.Flags().IntVar(&pingT.count, "ping-count", 5, "ping count")
	rootCmd.Flags().StringVar(&speedT.mode, "speed-test", "", "speed test mode")
	rootCmd.Flags().StringVar(&speedT.proxy, "speed-test-proxy", "", "speed test proxy")
	rootCmd.Flags().StringVar(&speedT.testUrl, "speed-test-url", "", "speed test url")
}

func initConfig() {
	// overwrite default options if use config file
	if configFile != "" {
		var err error
		jsonCfg, _, err = config.ParseJsonConfigFile(configFile)
		if err != nil {
			logx.FatalBy(err)
		}
	}

	if !jsonCfg.Verbose {
		logx.SetOutput(io.Discard)
	}
	if !jsonCfg.Color {
		logx.DisableColor = true
	}
	switch jsonCfg.VerboseLevel {
	case 1:
		logx.SetFlags(logx.FlagPrefix)
	case 2:
		logx.SetFlags(logx.FlagPrefix | logx.FlagTime | logx.FlagLineNumber)
	case 3:
		logx.SetFlags(logx.StdLoggerFlags)
	default:
		logx.SetFlags(logx.FlagPrefix | logx.FlagTime | logx.FlagLineNumber)
	}
}
