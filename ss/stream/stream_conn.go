package stream

import (
	"crypto/cipher"
	"crypto/rand"
	"io"
	"net"

	cipherx "github.com/josexy/mini-ss/cipher"
)

// uint16 using 2 bytes to store payload size
const maxPayloadBufferSize = 16 * 1024

type streamReader struct {
	net.Conn
	cipher.Stream
	buf []byte
}

func newStreamReader(c net.Conn, cipher cipher.Stream) *streamReader {
	return &streamReader{
		Conn:   c,
		Stream: cipher,
		buf:    make([]byte, maxPayloadBufferSize),
	}
}

func (r *streamReader) Read(b []byte) (int, error) {
	n, err := r.Conn.Read(b)
	if err != nil {
		return 0, err
	}
	r.XORKeyStream(b, b[:n])
	return n, err
}

func (r *streamReader) WriteTo(w io.Writer) (n int64, err error) {
	buf := r.buf[:]
	for {
		nr, er := r.Conn.Read(buf)
		if nr > 0 {
			r.XORKeyStream(buf, buf[:nr])
			nw, ew := w.Write(buf[:nr])
			n += int64(nw)
			if ew != nil {
				err = ew
				return
			}
		}
		if er != nil {
			if er != io.EOF { // ignore EOF as per io.Copy contract (using src.WriteTo shortcut)
				err = er
			}
			return
		}
	}
}

type streamWriter struct {
	net.Conn
	cipher.Stream
	buf []byte
}

func newStreamWriter(c net.Conn, cipher cipher.Stream) *streamWriter {
	return &streamWriter{
		Conn:   c,
		Stream: cipher,
		buf:    make([]byte, maxPayloadBufferSize),
	}
}

func (w *streamWriter) Write(p []byte) (int, error) {
	var n int
	var err error
	buf := w.buf[:]
	for nw := 0; n < len(p) && err == nil; n += nw {
		end := n + len(buf)
		if end > len(p) {
			end = len(p)
		}
		w.XORKeyStream(buf, p[n:end])
		nw, err = w.Conn.Write(buf[:end-n])
	}
	return n, err
}

func (w *streamWriter) ReadFrom(r io.Reader) (n int64, err error) {
	buf := w.buf[:]
	for {
		nr, er := r.Read(buf)
		n += int64(nr)
		b := buf[:nr]
		w.XORKeyStream(b, b)
		if _, err = w.Conn.Write(b); err != nil {
			return
		}
		if er != nil {
			if er != io.EOF { // ignore EOF as per io.ReaderFrom contract
				err = er
			}
			return
		}
	}
}

type StreamConn struct {
	net.Conn
	cipher  cipherx.StreamCipher
	r       *streamReader
	w       *streamWriter
	readIV  []byte
	writeIV []byte
}

func NewStreamConn(c net.Conn, cipher cipherx.StreamCipher) *StreamConn {
	return &StreamConn{
		Conn:   c,
		cipher: cipher,
	}
}

func (c *StreamConn) initReader() error {
	iv, err := c.ObtainReadIV()
	if err != nil {
		return err
	}
	// init decrypter
	cp, err := c.cipher.Decrypter(iv)
	if err != nil {
		return err
	}
	c.r = newStreamReader(c.Conn, cp)
	return nil
}

func (c *StreamConn) ObtainWriteIV() ([]byte, error) {
	if len(c.writeIV) == c.cipher.IVSize() {
		return c.writeIV, nil
	}
	iv := make([]byte, c.cipher.IVSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	c.writeIV = iv
	return iv, nil
}

func (c *StreamConn) ObtainReadIV() ([]byte, error) {
	if len(c.readIV) == c.cipher.IVSize() {
		return c.readIV, nil
	}
	iv := make([]byte, c.cipher.IVSize())
	_, err := io.ReadFull(c.Conn, iv)
	if err != nil {
		return nil, err
	}

	c.readIV = iv
	return iv, nil
}

func (c *StreamConn) initWriter() error {
	iv, err := c.ObtainWriteIV()
	if err != nil {
		return err
	}
	if _, err := c.Conn.Write(iv); err != nil {
		return err
	}
	cp, err := c.cipher.Encrypter(iv)
	if err != nil {
		return err
	}
	c.w = newStreamWriter(c.Conn, cp)
	return nil
}

func (c *StreamConn) Read(b []byte) (int, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.Read(b)
}

func (c *StreamConn) Write(b []byte) (int, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.Write(b)
}

func (c *StreamConn) WriteTo(w io.Writer) (int64, error) {
	if c.r == nil {
		if err := c.initReader(); err != nil {
			return 0, err
		}
	}
	return c.r.WriteTo(w)
}

func (c *StreamConn) ReadFrom(r io.Reader) (int64, error) {
	if c.w == nil {
		if err := c.initWriter(); err != nil {
			return 0, err
		}
	}
	return c.w.ReadFrom(r)
}
