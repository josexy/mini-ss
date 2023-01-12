package ssr

import (
	"net"

	"github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/ssr/obfs"
	"github.com/josexy/mini-ss/ssr/protocol"
	"github.com/josexy/mini-ss/util"
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

	host, port, _ := net.SplitHostPort(addr)
	obfs, overHead, err := obfs.GetObfs(obfsName, &obfs.Base{
		Host:   host,
		Port:   util.MustStringToInt(port),
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
