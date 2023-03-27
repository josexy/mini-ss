package transport

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/gorilla/websocket"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/util/logger"
	"github.com/quic-go/quic-go"
)

var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

type Conn interface {
	net.Conn
	TCP() *net.TCPConn
	UDP() *net.UDPConn
}

type TcpConnBound interface {
	TcpConn(net.Conn) net.Conn
}

type UdpConnBound interface {
	UdpConn(net.PacketConn) net.PacketConn
}

type TcpConnBoundHandler func(net.Conn) net.Conn

func (f TcpConnBoundHandler) TcpConn(c net.Conn) net.Conn { return f(c) }

type UdpConnBoundHandler func(net.PacketConn) net.PacketConn

func (f UdpConnBoundHandler) UdpConn(c net.PacketConn) net.PacketConn { return f(c) }

type SSPacketConn struct {
	net.PacketConn
	addr net.Addr
}

func (c *SSPacketConn) WriteTo(b []byte, _ net.Addr) (n int, err error) {
	return c.PacketConn.WriteTo(b, c.addr)
}

type CompressConn struct {
	net.Conn
	r *snappy.Reader
	w *snappy.Writer
}

func NewCompressConn(c net.Conn) *CompressConn {
	return &CompressConn{
		Conn: c,
		r:    snappy.NewReader(c),
		w:    snappy.NewBufferedWriter(c),
	}
}

func (c *CompressConn) Write(b []byte) (int, error) { return c.w.Write(b) }

func (c *CompressConn) Read(b []byte) (int, error) { return c.r.Read(b) }

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

type WebsocketConn struct {
	conn *websocket.Conn
	rbuf []byte // remaining buffer data
}

func NewWebsocketConn(c *websocket.Conn) *WebsocketConn { return &WebsocketConn{conn: c} }

func (c *WebsocketConn) Read(b []byte) (int, error) {
	var err error
	if len(c.rbuf) == 0 {
		_, c.rbuf, err = c.conn.ReadMessage()
	}
	n := copy(b, c.rbuf)
	c.rbuf = c.rbuf[n:]
	return n, err
}

func (c *WebsocketConn) Write(b []byte) (int, error) {
	return len(b), c.conn.WriteMessage(websocket.BinaryMessage, b)
}

func (c *WebsocketConn) Close() error { return c.conn.Close() }

func (c *WebsocketConn) LocalAddr() net.Addr { return c.conn.LocalAddr() }

func (c *WebsocketConn) RemoteAddr() net.Addr { return c.conn.RemoteAddr() }

func (c *WebsocketConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *WebsocketConn) SetReadDeadline(t time.Time) error { return c.conn.SetReadDeadline(t) }

func (c *WebsocketConn) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }

type ObfsConn struct {
	net.Conn
	host          string
	rbuf          bytes.Buffer
	wbuf          bytes.Buffer
	isServer      bool
	headerDrained bool
	handshakeMux  sync.Mutex
	handshaked    bool
}

func NewObfsConn(c net.Conn, host string, server bool) *ObfsConn {
	return &ObfsConn{
		Conn:     c,
		host:     host,
		isServer: server,
	}
}

// serverHandshake
/*
GET / HTTP/1.1
Host: www.google.com
User-Agent: Chrome/78.0.3904.106
Connection: Upgrade
Sec-Websocket-Key: nzrTjGlHVeXxadrEsn9bVQ==
Upgrade: websocket

// HTTP body
030d www.baidu.com
GET / HTTP/1.1
Host: www.baidu.com
User-Agent: curl/7.86.0

// SOCKS body
01 xxxxxx:80
030d www.baidu.com
GET / HTTP/1.1
Host: www.baidu.com
User-Agent: curl/7.86.0
*/
func (c *ObfsConn) serverHandshake() (err error) {
	br := bufio.NewReader(c.Conn)
	// GET
	r, err := http.ReadRequest(br)
	if err != nil {
		return err
	}

	// if the request has a body, read all headers and body at once
	if r.ContentLength > 0 {
		_, err = io.Copy(&c.rbuf, r.Body)
	} else {
		var b []byte
		b, err = br.Peek(br.Buffered())
		if len(b) > 0 {
			_, err = c.rbuf.Write(b)
		}
	}
	if err != nil {
		return
	}

	var b bytes.Buffer

	host := r.Host
	if host == "" && r.URL != nil {
		host = r.URL.Host
	}

	// check method
	// check header host
	if r.Method != http.MethodGet ||
		r.Header.Get("Upgrade") != "websocket" || host != c.host {
		b.WriteString("HTTP/1.1 503 Service Unavailable\r\n")
		b.WriteString("Content-Length: 0\r\n")
		b.WriteString("Date: " + time.Now().Format(time.RFC1123) + "\r\n")
		b.WriteString("\r\n")

		b.WriteTo(c.Conn)
		return errors.New("bad request")
	}

	/*
		HTTP/1.1 101 Switching Protocols
		Server: nginx/1.10.0
		Date: Wed, 09 Nov 2022 23:03:15 CST
		Connection: Upgrade
		Upgrade: websocket
		Sec-WebSocket-Accept: RAc75y1AdIadzgbuIcYxyyNERxs=

		......
	*/
	b.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	b.WriteString("Server: nginx/1.10.0\r\n")
	b.WriteString("Date: " + time.Now().Format(time.RFC1123) + "\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n", computeAcceptKey(r.Header.Get("Sec-WebSocket-Key"))))
	b.WriteString("\r\n")

	if c.rbuf.Len() > 0 {
		c.wbuf = b
		return
	}
	_, err = b.WriteTo(c.Conn)
	return
}

func computeAcceptKey(challengeKey string) string {
	h := sha1.New()
	h.Write([]byte(challengeKey))
	h.Write(keyGUID)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func generateChallengeKey() (string, error) {
	p := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(p), nil
}

func (c *ObfsConn) clientHandshake() error {
	req := &http.Request{
		Method:     http.MethodGet,
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL:        &url.URL{Scheme: "http", Host: c.host},
		Header:     make(http.Header),
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	key, _ := generateChallengeKey()
	req.Header.Set("Sec-WebSocket-Key", key)

	if err := req.Write(&c.wbuf); err != nil {
		return err
	}

	return nil
}

func (c *ObfsConn) Handshake() (err error) {
	c.handshakeMux.Lock()
	defer c.handshakeMux.Unlock()

	if c.handshaked {
		return nil
	}

	if !c.isServer {
		// client handshake
		err = c.clientHandshake()
	} else {
		// server handshake
		err = c.serverHandshake()
	}
	if err != nil {
		return
	}
	c.handshaked = true
	return
}

func (c *ObfsConn) drainHeader() (err error) {
	if c.headerDrained {
		return nil
	}
	c.headerDrained = true

	br := bufio.NewReader(c.Conn)
	var line string
	for {
		line, err = br.ReadString('\n')
		if err != nil {
			return
		}
		if line == "\r\n" {
			break
		}
	}
	// read remaining payload data
	var b []byte
	b, err = br.Peek(br.Buffered())
	if len(b) > 0 {
		_, err = c.rbuf.Write(b)
	}
	return
}

func (c *ObfsConn) Read(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}

	if !c.isServer {
		if err = c.drainHeader(); err != nil {
			return
		}
	}
	if c.rbuf.Len() > 0 {
		return c.rbuf.Read(b)
	}
	return c.Conn.Read(b)
}

// Write HTTP/SOCKS payload data into http request body section
func (c *ObfsConn) Write(b []byte) (n int, err error) {
	if err = c.Handshake(); err != nil {
		return
	}
	if c.wbuf.Len() > 0 {
		c.wbuf.Write(b)
		_, err = c.wbuf.WriteTo(c.Conn)
		n = len(b)
		return
	}
	return c.Conn.Write(b)
}
