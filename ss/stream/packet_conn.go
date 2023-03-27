package stream

import (
	"crypto/rand"
	"io"
	"net"

	cipherx "github.com/josexy/mini-ss/cipher"
	"github.com/josexy/mini-ss/constant"
)

type packetConn struct {
	net.PacketConn
	cipher cipherx.StreamCipher
	buf    []byte
}

func NewPacketConn(c net.PacketConn, cipher cipherx.StreamCipher) *packetConn {
	return &packetConn{
		PacketConn: c,
		cipher:     cipher,
		buf:        make([]byte, cipher.IVSize()+constant.MaxUdpBufferSize),
	}
}

func (c *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	ivLen := c.cipher.IVSize()
	buf := c.buf[:]
	if _, err := io.ReadFull(rand.Reader, buf[:ivLen]); err != nil {
		return 0, err
	}
	encCipher, err := c.cipher.Encrypter(buf[:ivLen])
	if err != nil {
		return 0, err
	}
	dataBuf := buf[ivLen:]
	encCipher.XORKeyStream(dataBuf, b)

	n := len(b)
	_, err = c.PacketConn.WriteTo(buf[:ivLen+n], addr)
	if err != nil {
		return 0, err
	}
	// written data bytes len(b)
	return n, nil
}

func (c *packetConn) ReadFrom(b []byte) (int, net.Addr, error) {
	buf := c.buf[:]
	n, addr, err := c.PacketConn.ReadFrom(buf)
	if err != nil {
		return n, addr, err
	}
	ivLen := c.cipher.IVSize()
	n = n - ivLen
	decCipher, err := c.cipher.Decrypter(buf[:ivLen])
	if err != nil {
		return n, addr, err
	}
	dataBuf := buf[ivLen:]
	decCipher.XORKeyStream(dataBuf[:n], dataBuf[:n])
	copy(b, dataBuf)
	// read data bytes
	return n, addr, nil
}
