package aead

import (
	"crypto/rand"
	"io"
	"net"

	cipherx "github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/constant"
)

var _zerononce [128]byte // read-only. 128 bytes is more than enough.

type packetConn struct {
	net.PacketConn
	cipher cipherx.AEADCipher
	buf    []byte
}

func NewPacketConn(c net.PacketConn, cipher cipherx.AEADCipher) *packetConn {
	return &packetConn{
		PacketConn: c,
		cipher:     cipher,
		buf:        make([]byte, constant.MaxUdpBufferSize),
	}
}

func (c *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	buf := c.buf[:]
	// [salt] [data]
	saltLen := c.cipher.SaltSize()
	if _, err := io.ReadFull(rand.Reader, buf[:saltLen]); err != nil {
		return 0, err
	}
	aead, err := c.cipher.Encrypter(buf[:saltLen])
	if err != nil {
		return 0, err
	}
	dataBuf := buf[saltLen:]
	res := aead.Seal(dataBuf[:0], _zerononce[:aead.NonceSize()], b, nil)

	_, err = c.PacketConn.WriteTo(buf[:saltLen+len(res)], addr)
	if err != nil {
		return 0, err
	}
	return len(b), err
}

func (c *packetConn) ReadFrom(b []byte) (int, net.Addr, error) {
	buf := c.buf[:]
	n, addr, err := c.PacketConn.ReadFrom(buf)
	if err != nil {
		return n, addr, err
	}
	saltLen := c.cipher.SaltSize()
	aead, err := c.cipher.Decrypter(buf[:saltLen])
	if err != nil {
		return n, addr, err
	}
	res, err := aead.Open(buf[saltLen:saltLen], _zerononce[:aead.NonceSize()], buf[saltLen:n], nil)
	if err != nil {
		return n, addr, err
	}
	copy(b, res)
	return len(res), addr, err
}
