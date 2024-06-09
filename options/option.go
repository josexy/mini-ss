package options

import (
	"crypto/tls"
	"time"

	"github.com/josexy/mini-ss/util/cert"
)

type TlsMode byte

const (
	None TlsMode = iota
	TLS
	MTLS
)

type TlsOptions struct {
	Mode     TlsMode
	CAFile   string
	KeyFile  string // server or client key file
	CertFile string // server or client cert file
	Hostname string
}

func (o *TlsOptions) GetServerTlsConfig() (*tls.Config, error) {
	var tlsConfig *tls.Config
	var err error
	switch o.Mode {
	case TLS:
		tlsConfig, err = cert.GetServerTlsConfig(o.CertFile, o.KeyFile)
	case MTLS:
		tlsConfig, err = cert.GetServerMTlsConfig(o.CertFile, o.KeyFile, o.CAFile)
	}
	return tlsConfig, err
}

func (o *TlsOptions) GetClientTlsConfig() (*tls.Config, error) {
	var tlsConfig *tls.Config
	var err error
	switch o.Mode {
	case TLS:
		tlsConfig, err = cert.GetClientTlsConfig(o.CAFile, o.Hostname)
	case MTLS:
		tlsConfig, err = cert.GetClientMTlsConfig(o.CertFile, o.KeyFile, o.CAFile, o.Hostname)
	}
	return tlsConfig, err
}

type Options interface{ Update() }

type defaultOptions struct {
	OutboundInterface   string
	AutoDetectInterface bool
}

func (defaultOptions) Update() {}

var DefaultOptions = &defaultOptions{}

var DefaultQuicOptions = &QuicOptions{
	HandshakeIdleTimeout: 5 * time.Second,
	KeepAlivePeriod:      30 * time.Second,
	MaxIdleTimeout:       30 * time.Second,
	Conns:                3,
}

var DefaultWsOptions = &WsOptions{
	Host:       "www.baidu.com",
	Path:       "/ws",
	SndBuffer:  4096,
	RevBuffer:  4096,
	Compress:   false,
	TlsOptions: TlsOptions{Mode: None},
	UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36",
}

var DefaultObfsOptions = &ObfsOptions{
	Host: "www.baidu.com",
}

var DefaultGrpcOptions = &GrpcOptions{
	TlsOptions: TlsOptions{Mode: None},
}

var DefaultSshOptions = &SshOptions{}

type WsOptions struct {
	TlsOptions
	Host      string
	Path      string
	SndBuffer int
	RevBuffer int
	Compress  bool
	UserAgent string
}

func (opts *WsOptions) Update() {}

type ObfsOptions struct {
	Host string
}

func (opts *ObfsOptions) Update() {}

type QuicOptions struct {
	TlsOptions
	HandshakeIdleTimeout time.Duration
	KeepAlivePeriod      time.Duration
	MaxIdleTimeout       time.Duration
	Conns                int
}

func (opts *QuicOptions) Update() {}

type GrpcOptions struct {
	TlsOptions
	SndBuffer int
	RevBuffer int
}

func (opts *GrpcOptions) Update() {}

type SshOptions struct {
	User          string
	Password      string
	PrivateKey    string
	PublicKey     string // only used for client
	AuthorizedKey string // only used for server
}

func (opts *SshOptions) Update() {}
