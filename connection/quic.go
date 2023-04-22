package connection

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"net"

	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/quic-go/quic-go"
)

type QuicConn struct {
	quic.Stream
	laddr net.Addr
	raddr net.Addr
}

func NewQuicConn(stream quic.Stream, laddr, raddr net.Addr) *QuicConn {
	return &QuicConn{
		Stream: stream,
		laddr:  laddr,
		raddr:  raddr,
	}
}

func (c *QuicConn) LocalAddr() net.Addr { return c.laddr }

func (c *QuicConn) RemoteAddr() net.Addr { return c.raddr }

type QuicCipherConn struct {
	net.PacketConn
	key   []byte
	buf   []byte
	nonce []byte
	gcm   cipher.AEAD
}

func NewQuicCipherConn(conn net.PacketConn, key []byte) *QuicCipherConn {
	if key == nil {
		logger.Logger.Fatal("quic cipher key can not be nil")
	}
	return &QuicCipherConn{
		PacketConn: conn,
		key:        key,
	}
}

func (c *QuicCipherConn) initCipher() error {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil
	}
	c.gcm, err = cipher.NewGCM(block)
	if err != nil {
		return err
	}
	c.nonce = make([]byte, c.gcm.NonceSize())
	c.buf = make([]byte, constant.MaxUdpBufferSize+c.gcm.Overhead())
	return nil
}

func (c *QuicCipherConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(b)
	if err != nil {
		return
	}
	data, err := c.decrypt(b[:n])
	if err != nil {
		return
	}
	copy(b, data)
	return len(data), addr, nil
}

func (c *QuicCipherConn) WirteTo(b []byte, addr net.Addr) (n int, err error) {
	data, err := c.encrypt(b)
	if err != nil {
		return
	}
	_, err = c.PacketConn.WriteTo(data, addr)
	if err != nil {
		return
	}
	return len(data), nil
}

func (c *QuicCipherConn) encrypt(b []byte) ([]byte, error) {
	if c.gcm == nil {
		if err := c.initCipher(); err != nil {
			return nil, err
		}
	}
	if _, err := io.ReadFull(rand.Reader, c.nonce); err != nil {
		return nil, err
	}
	return c.gcm.Seal(c.buf[:0], c.nonce, b, nil), nil
}

func (c *QuicCipherConn) decrypt(b []byte) ([]byte, error) {
	if c.gcm == nil {
		if err := c.initCipher(); err != nil {
			return nil, err
		}
	}
	nc := c.gcm.NonceSize()
	if len(b) < nc {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := b[:nc], b[nc:]
	return c.gcm.Open(c.buf[:0], nonce, ciphertext, nil)
}
