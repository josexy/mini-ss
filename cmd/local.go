package cmd

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/enhancer"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/josexy/proxyutil"
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
	localCmd.Flags().BoolVar(&cfg.Server[0].Udp, "udp-relay", false, "enable udp relay for client SOCKS proxy")

	// ssr
	localCmd.Flags().StringVarP(&cfg.Server[0].Type, "type", "T", "", "enable shadowsocksr")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.Protocol, "ssr-protocol", "O", "origin", "ssr protocol plugin")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.ProtocolParam, "ssr-protocol-param", "G", "", "ssr protocol param")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.Obfs, "ssr-obfs", "o", "plain", "ssr obfs plugin")
	localCmd.Flags().StringVarP(&cfg.Server[0].SSR.ObfsParam, "ssr-obfs-param", "g", "", "ssr obfs param")

	// tun mode
	localCmd.Flags().BoolVar(&cfg.Local.EnableTun, "enable-tun", false, "enable the local tun device, administrator privileges are required")
	localCmd.Flags().StringVar(&cfg.Local.Tun.Name, "tun-name", "utun3", "tun interface name")
	localCmd.Flags().StringVar(&cfg.Local.Tun.Cidr, "tun-cidr", "198.18.0.1/16", "tun interface cidr")
	localCmd.Flags().IntVar(&cfg.Local.Tun.Mtu, "tun-mtu", enhancer.DefaultMTU, "tun interface mtu")

	// fake dns mode
	localCmd.Flags().StringVar(&cfg.Local.FakeDNS.Listen, "fake-dns-listen", ":53", "fake-dns listening address")
	localCmd.Flags().StringSliceVar(&cfg.Local.FakeDNS.Nameservers, "fake-dns-nameservers", dns.DefaultDnsNameservers, "fake-dns nameservers")

	// interface
	localCmd.PersistentFlags().StringVar(&cfg.Iface, "iface", "", "bind outbound interface")
	localCmd.PersistentFlags().BoolVar(&cfg.AutoDetectIface, "auto-detect-iface", false, "enable auto-detect interface")
}

func StartLocal() {
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
		if cfg.Local.EnableTun {
			dnsutil.UnsetLocalDnsServer()
		}
		if cfg.Local.SystemProxy {
			proxyutil.UnsetSystemProxy()
		}
	}()
	startLocal()
}

func startLocal() {
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
