package aead

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"net"

	"github.com/josexy/mini-ss/bufferpool"
	cipherx "github.com/josexy/mini-ss/cipher"
)

type streamReader struct {
	net.Conn
	cipher.AEAD
	buf       []byte
	remainbuf []byte // remaining buffer
	nonce     []byte
}

func newStreamReader(c net.Conn, cipher cipher.AEAD) *streamReader {
	return &streamReader{
		Conn:  c,
		AEAD:  cipher,
		buf:   make([]byte, bufferpool.MaxTcpBufferSize+cipher.Overhead()),
		nonce: make([]byte, cipher.NonceSize()),
	}
}

func (r *streamReader) read() (int, error) {
	// ciphertext storage buffer size = 2 + overhead()
	buf := r.buf[:2+r.Overhead()]
	_, err := io.ReadFull(r.Conn, buf)
	if err != nil {
		return 0, err
	}

	// decrypt the payload and tag size firstly
	_, err = r.Open(buf[:0], r.nonce, buf, nil)
	increment(r.nonce)
	if err != nil {
		return 0, err
	}

	// n is the payload size
	size := int(buf[0])<<8 | int(buf[1]&0xFF)
	if size > bufferpool.MaxTcpBufferSize+r.Overhead() {
		return 0, errors.New("payload buffer size overflow")
	}
	// reset buffer to store the payload and tag data
	buf = r.buf[:size+r.Overhead()]
	_, err = io.ReadFull(r.Conn, buf)
	if err != nil {
		return 0, err
	}
	// decrypt the payload and tag data finally
	_, err = r.Open(buf[:0], r.nonce, buf, nil)
	increment(r.nonce)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (r *streamReader) Read(b []byte) (int, error) {
	// whether the buffer has remaining data
	if len(r.remainbuf) > 0 {
		n := copy(b, r.remainbuf)
		r.remainbuf = r.remainbuf[n:]
		return n, nil
	}
	n, err := r.read()
	if err != nil {
		return 0, err
	}
	m := copy(b, r.buf[:n])
	if m < n {
		r.remainbuf = r.buf[m:n]
	}
	return m, nil
}

func (r *streamReader) WriteTo(w io.Writer) (n int64, err error) {
	for len(r.remainbuf) > 0 {
		nw, ew := w.Write(r.remainbuf)
		r.remainbuf = r.remainbuf[nw:]
		n += int64(nw)
		if ew != nil {
			return n, ew
		}
	}
	for {
		nr, er := r.read()
		if nr > 0 {
			nw, ew := w.Write(r.buf[:nr])
			// written bytes
			n += int64(nw)

			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return n, err
}

type streamWriter struct {
	net.Conn
	cipher.AEAD
	// { [payload-size] [tag size] } { [payload data] [tag data] }
	// ensure that the size of the buffer stored during encryption is sufficient
	buf   []byte
	nonce []byte
}

// newStreamWriter cipher.Overhead() indicates the maximum difference between plaintext and ciphertext lengths :16
func newStreamWriter(c net.Conn, cipher cipher.AEAD) *streamWriter {
	return &streamWriter{
		Conn: c,
		AEAD: cipher,
		// the payload size of 2 bytes is 2+cipher.Overhead() after encryption
		// the maximum size of encrypted payload data is maxPayloadBufferSize+cipher.Overhead()
		buf:   make([]byte, 2+cipher.Overhead()+bufferpool.MaxTcpBufferSize+cipher.Overhead()),
		nonce: make([]byte, cipher.NonceSize()), // nonce size
	}
}

func (w *streamWriter) Write(b []byte) (int, error) {
	n, err := w.ReadFrom(bytes.NewReader(b))
	return int(n), err
}

func (w *streamWriter) ReadFrom(r io.Reader) (n int64, err error) {
	for {
		// reset buffer
		buf := w.buf[:]
		// buffer to store ciphertext
		dataBuf := buf[2+w.Overhead() : 2+w.Overhead()+bufferpool.MaxTcpBufferSize]
		// store payload data into buffer[2+w.Overhead():]
		// the buf[0] and buf[1] store data size
		nr, er := r.Read(dataBuf)
		if nr > 0 {
			n += int64(nr)

			// payload size
			buf[0], buf[1] = byte(nr>>8), byte(nr&0xFF)

			// encrypt the payload size and firstly
			// => payload + tag size: 2+overhead()
			w.Seal(buf[:0], w.nonce, buf[:2], nil)
			increment(w.nonce)
			// encrypt the payload data finally
			// => payload + tag data: nr+overhead()
			w.Seal(dataBuf[:0], w.nonce, dataBuf[:nr], nil)
			increment(w.nonce)

			// the data bytes actually written: 2+overhead()+nr+overhead()
			_, ew := w.Conn.Write(buf[:2+w.Overhead()+nr+w.Overhead()])

			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return n, err
}

func increment(nonce []byte) {
	for i := range nonce {
		nonce[i]++
		if nonce[i] != 0 {
			return
		}
	}
}

type streamConn struct {
	net.Conn
	cipher cipherx.AEADCipher
	r      *streamReader
	w      *streamWriter
}

// NewStreamConn data format: { [salt data] } { [payload-size] [tag size] } { [payload data] [tag data] }
func NewStreamConn(c net.Conn, cipher cipherx.AEADCipher) *streamConn {
	return &streamConn{
		Conn:   c,
		cipher: cipher,
	}
}

func (c *streamConn) initReader() error {
	salt := make([]byte, c.cipher.SaltSize())
	_, err := io.ReadFull(c.Conn, salt)
	if err != nil {
		return err
	}
	// init decrypter
	cp, err := c.cipher.GetDecrypter(salt)
	if err != nil {
		return err
	}
	c.r = newStreamReader(c.Conn, cp)
	return nil
}

func (c *streamConn) initWriter() error {
	salt := make([]byte, c.cipher.SaltSize())
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	if _, err := c.Conn.Write(salt); err != nil {
		return err
	}
	cp, err := c.cipher.GetEncrypter(salt)
	if err != nil {
		return err
	}
	c.w = newStreamWriter(c.Conn, cp)
	return nil
}

func (c *streamConn) Read(b []byte) (int, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.Read(b)
}

func (c *streamConn) Write(b []byte) (int, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.Write(b)
}

func (c *streamConn) WriteTo(w io.Writer) (int64, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.WriteTo(w)
}

func (c *streamConn) ReadFrom(r io.Reader) (int64, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.ReadFrom(r)
}
