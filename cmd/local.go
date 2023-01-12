package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/dns"
	"github.com/josexy/mini-ss/geoip"
	"github.com/josexy/mini-ss/ss"
	"github.com/josexy/mini-ss/tun"
	"github.com/josexy/mini-ss/util/dnsutil"
	"github.com/josexy/mini-ss/util/proxyutil"
	"github.com/spf13/cobra"
)

var localCmd = &cobra.Command{
	Use:     "client",
	Short:   "ss-local subcommand options",
	Example: "  mini-ss client -s 127.0.0.1:8388 -l :10086 -x :10087 -m aes-128-cfb -p 123456 -CV",
	Run: func(cmd *cobra.Command, args []string) {
		defer func() {
			if err := recover(); err != nil {
				if e, ok := err.(error); ok {
					logx.FatalBy(e)
				}
			}
			if jsonCfg.Local.EnableTun {
				dnsutil.UnsetLocalDnsServer()
			}
			if jsonCfg.Local.SystemProxy {
				proxyutil.UnsetSystemProxy()
			}
		}()

		if len(jsonCfg.Server) > 0 && jsonCfg.Server[0].Addr == "" {
			cmd.Help()
			return
		}

		if err := startLocal(); err != nil {
			logx.FatalBy(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(localCmd)

	localCmd.Flags().StringVarP(&jsonCfg.Server[0].Addr, "server", "s", "", "client connects to server address")
	localCmd.Flags().StringVarP(&jsonCfg.Local.SocksAddr, "socks", "l", "127.0.0.1:10086", "SOCKS proxy listening address")
	localCmd.Flags().StringVarP(&jsonCfg.Local.HTTPAddr, "http", "x", "", "HTTP proxy listening address")
	localCmd.Flags().StringVar(&jsonCfg.Local.SocksAuth, "socks-auth", "", "SOCKS proxy authentication (format: \"user:password\")")
	localCmd.Flags().StringVar(&jsonCfg.Local.HTTPAuth, "http-auth", "", "HTTP proxy authentication (format: \"user:password\")")
	localCmd.Flags().StringVarP(&jsonCfg.Local.MixedAddr, "mixed", "M", "", "mixed proxy for SOCKS and HTTP")
	localCmd.Flags().StringSliceVar(&jsonCfg.Local.TCPTunAddr, "tcp-tun", nil, "simple tcp tun listening address (format: \"local:port=remote:port\")")
	localCmd.Flags().StringSliceVar(&jsonCfg.Local.UDPTunAddr, "udp-tun", nil, "simple udp tun listening address (format: \"local:port=remote:port\")")
	localCmd.Flags().BoolVar(&jsonCfg.Local.SystemProxy, "system-proxy", false, "enable system proxy settings")

	// ssr
	localCmd.Flags().StringVarP(&jsonCfg.Server[0].Type, "type", "T", "", "enable shadowsocksr")
	localCmd.Flags().StringVarP(&jsonCfg.Server[0].SSR.Protocol, "ssr-protocol", "O", "origin", "ssr protocol plugin")
	localCmd.Flags().StringVarP(&jsonCfg.Server[0].SSR.ProtocolParam, "ssr-protocol-param", "G", "", "ssr protocol param")
	localCmd.Flags().StringVarP(&jsonCfg.Server[0].SSR.Obfs, "ssr-obfs", "o", "plain", "ssr obfs plugin")
	localCmd.Flags().StringVarP(&jsonCfg.Server[0].SSR.ObfsParam, "ssr-obfs-param", "g", "", "ssr obfs param")

	// tun mode
	localCmd.Flags().BoolVar(&jsonCfg.Local.EnableTun, "enable-tun", false, "enable the local tun device, administrator privileges are required")
	localCmd.Flags().StringVar(&jsonCfg.Local.Tun.Name, "tun-name", "utun3", "tun interface name")
	localCmd.Flags().StringVar(&jsonCfg.Local.Tun.Cidr, "tun-cidr", "198.18.0.1/16", "tun interface cidr")
	localCmd.Flags().IntVar(&jsonCfg.Local.Tun.Mtu, "tun-mtu", tun.DefaultMTU, "tun interface mtu")

	// fake dns mode
	localCmd.Flags().StringVar(&jsonCfg.Local.FakeDNS.Listen, "fake-dns-listen", ":53", "fake-dns listening address")
	localCmd.Flags().StringSliceVar(&jsonCfg.Local.FakeDNS.Nameservers, "fake-dns-nameservers", dns.DefaultDnsNameservers, "fake-dns nameservers")
}

func startLocal() error {
	geoip.Data, _ = os.ReadFile("Country.mmdb")

	srv := ss.NewShadowsocksClient(jsonCfg.BuildSSLocalOptions()...)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := srv.Start(); err != nil {
		return err
	}
	<-interrupt

	srv.Close()
	return nil
}
