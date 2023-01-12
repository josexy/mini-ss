package constant

const (
	MaxBufferSize    = 515
	MaxUdpBufferSize = 1 << 13
)

const (
	Socks5Version05 = 0x05
	Socks5Version01 = 0x01
)

const (
	MethodNoAuthRequired   = 0x00
	MethodGSSAPI           = 0x01
	MethodUsernamePassword = 0x02
	MethodIANAAssigned     = 0x03
	MethodReserved         = 0x80
	MethodNotAcceptable    = 0xFF
)

type Socks5Cmd = byte

const (
	Connect byte = iota + 1
	Bind
	UDP
)

const (
	IPv4 = 0x01
	Fqdn = 0x03
	IPv6 = 0x04
)

const (
	Succeed byte = iota
	GeneralSocksServerFailure
	ConnectionNotAllowedByRuleset
	NetworkUnreachable
	HostUnreachable
	ConnectionRefused
	TTLExpired
	CommandNotSupported
	AddressTypeNotSupported
	Unassigned
)

// UDP relayer
const (
	UDPSSLocalToSocksClient = iota
	UDPSSServerToSSLocal
	UDPTunServerToUDPClient

	UDPSSLocalToSSServer
	UDPSSServerToUDPServer
	UDPTunServerToSSServer
)

// TCP relayer
const (
	TCPSSLocalToSSServer = iota
	TCPSSServerToTCPServer
)
