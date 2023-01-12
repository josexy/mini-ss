package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"net"
)

var (
	errAuthSHA1V4CRC32Error   = errors.New("auth_sha1_v4 decode data wrong crc32")
	errAuthSHA1V4LengthError  = errors.New("auth_sha1_v4 decode data wrong length")
	errAuthSHA1V4Adler32Error = errors.New("auth_sha1_v4 decode data wrong adler32")
	errAuthAES128MACError     = errors.New("auth_aes128 decode data wrong mac")
	errAuthAES128LengthError  = errors.New("auth_aes128 decode data wrong length")
	errAuthAES128ChksumError  = errors.New("auth_aes128 decode data wrong checksum")
	errAuthChainLengthError   = errors.New("auth_chain decode data wrong length")
	errAuthChainChksumError   = errors.New("auth_chain decode data wrong checksum")
)

const relayBufferSize = 20 * 1024

const (
	protocolOrigin         = "origin"
	protocolAuthAES128MD5  = "auth_aes128_md5"
	protocolAuthAES128SHA1 = "auth_aes128_sha1"
	protocolAuthChainA     = "auth_chain_a"
	protocolAuthChainB     = "auth_chain_b"
	protocolAuthSHA1V4     = "auth_sha1_v4"
)

type Protocol interface {
	StreamConn(net.Conn, []byte) net.Conn
	PacketConn(net.PacketConn) net.PacketConn
	Decode(dst, src *bytes.Buffer) error
	Encode(buf *bytes.Buffer, b []byte) error
	DecodePacket([]byte) ([]byte, error)
	EncodePacket(buf *bytes.Buffer, b []byte) error
}

type protocolWrapper struct {
	Overhead int
	New      func(*Base) Protocol
}

var protocolMap = map[string]protocolWrapper{
	protocolOrigin:         {0, newOrigin},
	protocolAuthAES128MD5:  {9, newAuthAES128MD5},
	protocolAuthAES128SHA1: {9, newAuthAES128SHA1},
	protocolAuthChainA:     {4, newAuthChainA},
	protocolAuthChainB:     {4, newAuthChainB},
	protocolAuthSHA1V4:     {7, newAuthSHA1V4},
}

func GetProtocol(name string, b *Base) (Protocol, error) {
	if prot, ok := protocolMap[name]; ok {
		b.Overhead += prot.Overhead
		return prot.New(b), nil
	}
	return nil, fmt.Errorf("protocol %s not supported", name)
}

func getHeadSize(b []byte, defaultValue int) int {
	if len(b) < 2 {
		return defaultValue
	}
	headType := b[0] & 7
	switch headType {
	case 1:
		return 7
	case 4:
		return 19
	case 3:
		return 4 + int(b[1])
	}
	return defaultValue
}

func getDataLength(b []byte) int {
	bLength := len(b)
	dataLength := getHeadSize(b, 30) + rand.Intn(32)
	if bLength < dataLength {
		return bLength
	}
	return dataLength
}
