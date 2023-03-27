package constant

import "time"

const (
	MaxSocksBufferSize = 515
	MaxUdpBufferSize   = 16 * 1024
	MaxTcpBufferSize   = 16 * 1024
	UdpTimeout         = 30 * time.Second
)

const (
	Connect byte = iota + 1
	UDP
	Bind
)
