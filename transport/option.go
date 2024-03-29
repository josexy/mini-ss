package transport

import (
	"crypto/sha1"
	"time"

	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

type defaultDialerOutboundOption struct {
	Interface           string
	AutoDetectInterface bool
}

var DefaultDialerOutboundOption = &defaultDialerOutboundOption{}

type Options interface{ Update() }

type defaultDialerOption struct{}

func (defaultDialerOption) Update() {}

var DefaultOptions = &defaultDialerOption{}

var DefaultKcpOptions = &KcpOptions{
	Crypt:       "none",
	Key:         "",
	Mode:        "normal",
	Mtu:         1350,
	SndWnd:      2048,
	RevWnd:      2048,
	DataShard:   10,
	ParityShard: 3,
	Dscp:        46,
	Resend:      2,
	NoCompress:  true,
	AckNoDelay:  false,
	Interval:    40,
	Nc:          1,
	SockBuf:     16777217,
	SmuxVer:     1,
	SmuxBuf:     16777217,
	StreamBuf:   2097152,
	KeepAlive:   10,
	Conns:       3,
}

var DefaultQuicOptions = &QuicOptions{
	HandshakeIdleTimeout: 5 * time.Second,
	KeepAlivePeriod:      30 * time.Second,
	MaxIdleTimeout:       30 * time.Second,
	Conns:                3,
}

var DefaultWsOptions = &WsOptions{
	Host:      "www.baidu.com",
	Path:      "/ws",
	SndBuffer: 4096,
	RevBuffer: 4096,
	Compress:  false,
	TLS:       false,
	UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36",
}

var DefaultObfsOptions = &ObfsOptions{
	Host: "www.baidu.com",
}

var DefaultGrpcOptions = &GrpcOptions{}

type KcpOptions struct {
	Key         string
	Crypt       string
	Mode        string
	Mtu         int
	SndWnd      int
	RevWnd      int
	DataShard   int
	ParityShard int
	Dscp        int
	NoCompress  bool
	AckNoDelay  bool
	NoDelay     int
	Interval    int
	Resend      int
	Nc          int
	SockBuf     int
	SmuxVer     int
	SmuxBuf     int
	StreamBuf   int
	KeepAlive   int
	Conns       int

	BC kcp.BlockCrypt
}

func (opts *KcpOptions) Update() {
	switch opts.Mode {
	case "normal":
		opts.NoDelay, opts.Interval, opts.Resend, opts.Nc = 0, 40, 2, 1
	case "fast":
		opts.NoDelay, opts.Interval, opts.Resend, opts.Nc = 0, 30, 2, 1
	case "fast2":
		opts.NoDelay, opts.Interval, opts.Resend, opts.Nc = 1, 20, 2, 1
	case "fast3":
		opts.NoDelay, opts.Interval, opts.Resend, opts.Nc = 1, 10, 2, 1
	}
	opts.BC = kcpBlockCrypt(opts.Key, opts.Crypt, "mini-ss")
}

func kcpBlockCrypt(key, crypt, salt string) (block kcp.BlockCrypt) {
	pass := pbkdf2.Key([]byte(key), []byte(salt), 4096, 32, sha1.New)
	switch crypt {
	case "sm4":
		block, _ = kcp.NewSM4BlockCrypt(pass[:16])
	case "tea":
		block, _ = kcp.NewTEABlockCrypt(pass[:16])
	case "xor":
		block, _ = kcp.NewSimpleXORBlockCrypt(pass)
	case "aes-128":
		block, _ = kcp.NewAESBlockCrypt(pass[:16])
	case "aes-192":
		block, _ = kcp.NewAESBlockCrypt(pass[:24])
	case "aes-256":
		block, _ = kcp.NewAESBlockCrypt(pass[:32])
	case "blowfish":
		block, _ = kcp.NewBlowfishBlockCrypt(pass)
	case "twofish":
		block, _ = kcp.NewTwofishBlockCrypt(pass)
	case "cast5":
		block, _ = kcp.NewCast5BlockCrypt(pass[:16])
	case "3des":
		block, _ = kcp.NewTripleDESBlockCrypt(pass[:24])
	case "xtea":
		block, _ = kcp.NewXTEABlockCrypt(pass[:16])
	case "salsa20":
		block, _ = kcp.NewSalsa20BlockCrypt(pass)
	default:
		block, _ = kcp.NewNoneBlockCrypt(pass)
	}
	return
}

type WsOptions struct {
	Host      string
	Path      string
	SndBuffer int
	RevBuffer int
	Compress  bool
	UserAgent string
	TLS       bool // support tls
}

func (opts *WsOptions) Update() {}

type ObfsOptions struct {
	Host string
}

func (opts *ObfsOptions) Update() {}

type QuicOptions struct {
	HandshakeIdleTimeout time.Duration
	KeepAlivePeriod      time.Duration
	MaxIdleTimeout       time.Duration
	Conns                int
}

func (opts *QuicOptions) Update() {}

type GrpcOptions struct {
	SndBuffer int
	RevBuffer int
	TLS       bool
	Hostname  string
	CAPath    string
	CertPath  string
	KeyPath   string
}

func (opts *GrpcOptions) Update() {}
