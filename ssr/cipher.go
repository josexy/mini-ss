package ssr

import (
	"net"
	"strconv"

	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/ssr/obfs"
	"github.com/josexy/mini-ss/ssr/protocol"
)

// SSRClientStreamCipher shadowsocks StreamCipher wrapper key
type SSRClientStreamCipher struct {
	cipher.StreamCipher
	Obfs  obfs.Obfs
	Proto protocol.Protocol
}

func NewSSRClientStreamCipher(ciph cipher.StreamCipher, addr string,
	protocolName, protocolParam,
	obfsName, obfsParam string) (*SSRClientStreamCipher, error) {

	var key []byte
	var ivSize int
	if ciph != nil {
		key = ciph.Key()
		ivSize = ciph.IVSize()
	}

	host, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.ParseUint(p, 10, 16)
	obfs, overHead, err := obfs.GetObfs(obfsName, &obfs.Base{
		Host:   host,
		Port:   int(port),
		Key:    key,
		IVSize: ivSize,
		Param:  obfsParam,
	})
	if err != nil {
		return nil, err
	}
	proto, err := protocol.GetProtocol(protocolName, &protocol.Base{
		Key:      key,
		Overhead: overHead,
		Param:    protocolParam,
	})
	if err != nil {
		return nil, err
	}

	return &SSRClientStreamCipher{
		StreamCipher: ciph,
		Obfs:         obfs,
		Proto:        proto,
	}, nil
}
