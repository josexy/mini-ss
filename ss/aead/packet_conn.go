package aead

import (
	"crypto/rand"
	"io"
	"net"

	"github.com/josexy/mini-ss/bufferpool"
	cipherx "github.com/josexy/mini-ss/cipher"
)

var _zerononce [128]byte // read-only. 128 bytes is more than enough.

type packetConn struct {
	net.PacketConn
	cipher cipherx.AEADCipher
	buf    []byte
}

// NewPacketConn data format: { [salt data] } { [payload data] [tag data] }
func NewPacketConn(c net.PacketConn, cipher cipherx.AEADCipher) *packetConn {
	return &packetConn{
		PacketConn: c,
		cipher:     cipher,
		buf:        make([]byte, bufferpool.MaxUdpBufferSize),
	}
}

func (c *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	buf := c.buf[:]
	// [salt] [data]
	saltLen := c.cipher.SaltSize()
	if _, err := io.ReadFull(rand.Reader, buf[:saltLen]); err != nil {
		return 0, err
	}
	aead, err := c.cipher.GetEncrypter(buf[:saltLen])
	if err != nil {
		return 0, err
	}
	if len(buf) < saltLen+len(b)+aead.Overhead() {
		return 0, io.ErrShortBuffer
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
	n, addr, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	saltLen := c.cipher.SaltSize()
	// short buffer and the stored salt data not enough
	if n < saltLen {
		return n, addr, io.ErrShortBuffer
	}
	dst := b[saltLen:]
	aead, err := c.cipher.GetDecrypter(b[:saltLen])
	if err != nil {
		return n, addr, err
	}
	dataLen := n - (saltLen + aead.Overhead())
	if dataLen < 0 || dataLen > len(dst) {
		return n, addr, io.ErrShortBuffer
	}
	res, err := aead.Open(dst[:0], _zerononce[:aead.NonceSize()], b[saltLen:n], nil)
	if err != nil {
		return n, addr, err
	}
	copy(b, res)
	return len(res), addr, err
}
