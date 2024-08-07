package cmd

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/enhancer"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/resolver"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/spf13/cobra"
)

var localCmd = &cobra.Command{
	Use:     "client",
	Short:   "ss-local subcommand options",
	Example: "  mini-ss client -s 127.0.0.1:8388 -l :10086 -x :10087 -m aes-128-cfb -p 123456 -CV3",
	Run: func(cmd *cobra.Command, args []string) {
		if (len(cfg.Server) == 0 || cfg.Server[0].Addr == "") && configFile == "" {
			cmd.Help()
			return
		}
		StartLocal()
	},
}

func init() {
	rootCmd.AddCommand(localCmd)

	localCmd.Flags().StringVarP(&cfg.Server[0].Addr, "server", "s", "", "client connects to server address")
	localCmd.Flags().StringVarP(&cfg.Local.SocksAddr, "socks", "l", "127.0.0.1:10086", "SOCKS proxy listening address")
	localCmd.Flags().StringVarP(&cfg.Local.HTTPAddr, "http", "x", "", "HTTP proxy listening address")
	localCmd.Flags().StringVar(&cfg.Local.SocksAuth, "socks-auth", "", "SOCKS proxy authentication (format: \"user:password\")")
	localCmd.Flags().StringVar(&cfg.Local.HTTPAuth, "http-auth", "", "HTTP proxy authentication (format: \"user:password\")")
	localCmd.Flags().StringVarP(&cfg.Local.MixedAddr, "mixed", "M", "", "mixed proxy for SOCKS and HTTP")
	localCmd.Flags().StringSliceVar(&cfg.Local.TCPTunAddr, "tcp-tun", nil, "simple tcp tun listening address (format: \"local:port=remote:port\")")
	localCmd.Flags().BoolVar(&cfg.Local.SystemProxy, "system-proxy", false, "enable system proxy settings")
	localCmd.Flags().BoolVar(&cfg.Local.LookupHostsFile, "lookup-hostsfile", false, "dns lookup local hosts file")

	// ssr
	localCmd.Flags().StringVarP(&cfg.Server[0].Type, "type", "T", "", "enable shadowsocksr")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.Protocol, "ssr-protocol", "O", "origin", "ssr protocol plugin")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.ProtocolParam, "ssr-protocol-param", "G", "", "ssr protocol param")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.Obfs, "ssr-obfs", "o", "plain", "ssr obfs plugin")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.ObfsParam, "ssr-obfs-param", "g", "", "ssr obfs param")

	// tun mode
	localCmd.Flags().BoolVar(&cfg.Local.Tun.Enable, "tun-enable", false, "enable the local tun device, administrator privileges are required")
	localCmd.Flags().StringVar(&cfg.Local.Tun.Name, "tun-name", "utun9", "tun interface name")
	localCmd.Flags().StringVar(&cfg.Local.Tun.Cidr, "tun-cidr", "198.18.0.1/16", "tun interface cidr")
	localCmd.Flags().IntVar(&cfg.Local.Tun.Mtu, "tun-mtu", enhancer.DefaultMTU, "tun interface mtu")
	localCmd.Flags().StringSliceVar(&cfg.Local.Tun.DnsHijack, "tun-dns-hijack", nil, "tun dns hijack")
	localCmd.Flags().BoolVar(&cfg.Local.Tun.AutoRoute, "tun-dns-auto-route", true, "tun auto route configured")

	// fake dns mode
	localCmd.Flags().StringVar(&cfg.Local.DNS.Listen, "fake-dns-listen", ":53", "fake-dns listening address")
	localCmd.Flags().StringSliceVar(&cfg.Local.DNS.Nameservers, "fake-dns-nameservers", resolver.DefaultDnsNameservers, "fake-dns nameservers")
	localCmd.Flags().StringSliceVar(&cfg.Local.DNS.DomainFilter, "fake-dns-domain-filter", nil, "fake-dns domain filter")
	localCmd.Flags().BoolVar(&cfg.Local.DNS.DisableRewrite, "fake-dns-disable-rewrite", false, "fake-dns disable to rewrite dns to system config file")

	// mitm mode
	localCmd.Flags().BoolVar(&cfg.Local.Mitm.Enable, "mitm-mode", false, "enable mitm mode")
	localCmd.Flags().StringVar(&cfg.Local.Mitm.CAPath, "mitm-ca-path", "", "mitm proxy ca cert path")
	localCmd.Flags().StringVar(&cfg.Local.Mitm.KeyPath, "mitm-key-path", "", "mitm proxy key path")
}

func StartLocal() {
	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(error); ok {
				logger.Logger.FatalBy(e)
			}
		}
	}()
	startLocal()
}

func startLocal() {
	logger.Logger.Info("build info", logx.String("version", Version), logx.String("git_commit", GitCommit))
	if err := geoip.OpenDB("Country.mmdb"); err != nil {
		logger.Logger.FatalBy(err)
		return
	}

	srv := ss.NewShadowsocksClient(cfg.BuildSSLocalOptions()...)

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
