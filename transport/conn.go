package transport

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	dissector "github.com/go-gost/tls-dissector"
	"github.com/golang/snappy"
	"github.com/gorilla/websocket"
	"github.com/josexy/logx"
	"github.com/quic-go/quic-go"
)

const (
	maxTLSDataLen    = 16384
	maxUDPPacketSize = 16 * 1024
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
		logx.Fatal("quic cipher key can not be nil")
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
	c.buf = make([]byte, maxUDPPacketSize+c.gcm.Overhead())
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

var (
	cipherSuites = []uint16{
		0xc02c, 0xc030, 0x009f, 0xcca9, 0xcca8, 0xccaa, 0xc02b, 0xc02f,
		0x009e, 0xc024, 0xc028, 0x006b, 0xc023, 0xc027, 0x0067, 0xc00a,
		0xc014, 0x0039, 0xc009, 0xc013, 0x0033, 0x009d, 0x009c, 0x003d,
		0x003c, 0x0035, 0x002f, 0x00ff,
	}

	compressionMethods = []uint8{0x00}

	algorithms = []uint16{
		0x0601, 0x0602, 0x0603, 0x0501, 0x0502, 0x0503, 0x0401, 0x0402,
		0x0403, 0x0301, 0x0302, 0x0303, 0x0201, 0x0202, 0x0203,
	}

	tlsRecordTypes   = []uint8{0x16, 0x14, 0x16, 0x17}
	tlsVersionMinors = []uint8{0x01, 0x03, 0x03, 0x03}

	ErrBadType         = errors.New("bad type")
	ErrBadMajorVersion = errors.New("bad major version")
	ErrBadMinorVersion = errors.New("bad minor version")
	ErrMaxDataLen      = errors.New("bad tls data len")
)

const (
	tlsRecordStateType = iota
	tlsRecordStateVersion0
	tlsRecordStateVersion1
	tlsRecordStateLength0
	tlsRecordStateLength1
	tlsRecordStateData
)

type obfsTLSParser struct {
	step   uint8
	state  uint8
	length uint16
}

type ObfsTLSConn struct {
	net.Conn
	rbuf           bytes.Buffer
	wbuf           bytes.Buffer
	host           string
	isServer       bool
	handshaked     chan struct{}
	parser         *obfsTLSParser
	handshakeMutex sync.Mutex
}

func NewObfsTLSConn(c net.Conn, host string, server bool) *ObfsTLSConn {
	return &ObfsTLSConn{
		Conn:       c,
		host:       host,
		isServer:   server,
		handshaked: make(chan struct{}),
		parser:     &obfsTLSParser{},
	}
}

func (r *obfsTLSParser) Parse(b []byte) (int, error) {
	i := 0
	last := 0
	length := len(b)

	for i < length {
		ch := b[i]
		switch r.state {
		case tlsRecordStateType:
			if tlsRecordTypes[r.step] != ch {
				return 0, ErrBadType
			}
			r.state = tlsRecordStateVersion0
			i++
		case tlsRecordStateVersion0:
			if ch != 0x03 {
				return 0, ErrBadMajorVersion
			}
			r.state = tlsRecordStateVersion1
			i++
		case tlsRecordStateVersion1:
			if ch != tlsVersionMinors[r.step] {
				return 0, ErrBadMinorVersion
			}
			r.state = tlsRecordStateLength0
			i++
		case tlsRecordStateLength0:
			r.length = uint16(ch) << 8
			r.state = tlsRecordStateLength1
			i++
		case tlsRecordStateLength1:
			r.length |= uint16(ch)
			if r.step == 0 {
				r.length = 91
			} else if r.step == 1 {
				r.length = 1
			} else if r.length > maxTLSDataLen {
				return 0, ErrMaxDataLen
			}
			if r.length > 0 {
				r.state = tlsRecordStateData
			} else {
				r.state = tlsRecordStateType
				r.step++
			}
			i++
		case tlsRecordStateData:
			left := uint16(length - i)
			if left > r.length {
				left = r.length
			}
			if r.step >= 2 {
				skip := i - last
				copy(b[last:], b[i:length])
				length -= int(skip)
				last += int(left)
				i = last
			} else {
				i += int(left)
			}
			r.length -= left
			if r.length == 0 {
				if r.step < 3 {
					r.step++
				}
				r.state = tlsRecordStateType
			}
		}
	}

	if last == 0 {
		return 0, nil
	} else if last < length {
		length -= last
	}

	return length, nil
}

func (c *ObfsTLSConn) Handshaked() bool {
	select {
	case <-c.handshaked:
		return true
	default:
		return false
	}
}

func (c *ObfsTLSConn) Handshake(payload []byte) (err error) {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	if c.Handshaked() {
		return
	}

	if c.isServer {
		err = c.serverHandshake()
	} else {
		err = c.clientHandshake(payload)
	}
	if err != nil {
		return
	}

	close(c.handshaked)
	return nil
}

func (c *ObfsTLSConn) clientHandshake(payload []byte) error {
	clientMsg := &dissector.ClientHelloMsg{
		Version:            tls.VersionTLS12,
		SessionID:          make([]byte, 32),
		CipherSuites:       cipherSuites,
		CompressionMethods: compressionMethods,
		Extensions: []dissector.Extension{
			&dissector.SessionTicketExtension{
				Data: payload,
			},
			&dissector.ServerNameExtension{
				Name: c.host,
			},
			&dissector.ECPointFormatsExtension{
				Formats: []uint8{0x01, 0x00, 0x02},
			},
			&dissector.SupportedGroupsExtension{
				Groups: []uint16{0x001d, 0x0017, 0x0019, 0x0018},
			},
			&dissector.SignatureAlgorithmsExtension{
				Algorithms: algorithms,
			},
			&dissector.EncryptThenMacExtension{},
			&dissector.ExtendedMasterSecretExtension{},
		},
	}
	clientMsg.Random.Time = uint32(time.Now().Unix())
	rand.Read(clientMsg.Random.Opaque[:])
	rand.Read(clientMsg.SessionID)
	b, err := clientMsg.Encode()
	if err != nil {
		return err
	}

	record := &dissector.Record{
		Type:    dissector.Handshake,
		Version: tls.VersionTLS10,
		Opaque:  b,
	}
	if _, err := record.WriteTo(c.Conn); err != nil {
		return err
	}
	return err
}

func (c *ObfsTLSConn) serverHandshake() error {
	record := &dissector.Record{}
	if _, err := record.ReadFrom(c.Conn); err != nil {
		return err
	}
	if record.Type != dissector.Handshake {
		return dissector.ErrBadType
	}

	clientMsg := &dissector.ClientHelloMsg{}
	if err := clientMsg.Decode(record.Opaque); err != nil {
		return err
	}

	for _, ext := range clientMsg.Extensions {
		if ext.Type() == dissector.ExtSessionTicket {
			b, err := ext.Encode()
			if err != nil {
				return err
			}
			c.rbuf.Write(b)
			break
		}
	}

	serverMsg := &dissector.ServerHelloMsg{
		Version:           tls.VersionTLS12,
		SessionID:         clientMsg.SessionID,
		CipherSuite:       0xcca8,
		CompressionMethod: 0x00,
		Extensions: []dissector.Extension{
			&dissector.RenegotiationInfoExtension{},
			&dissector.ExtendedMasterSecretExtension{},
			&dissector.ECPointFormatsExtension{
				Formats: []uint8{0x00},
			},
		},
	}

	serverMsg.Random.Time = uint32(time.Now().Unix())
	rand.Read(serverMsg.Random.Opaque[:])
	b, err := serverMsg.Encode()
	if err != nil {
		return err
	}

	record = &dissector.Record{
		Type:    dissector.Handshake,
		Version: tls.VersionTLS10,
		Opaque:  b,
	}

	if _, err := record.WriteTo(&c.wbuf); err != nil {
		return err
	}

	record = &dissector.Record{
		Type:    dissector.ChangeCipherSpec,
		Version: tls.VersionTLS12,
		Opaque:  []byte{0x01},
	}
	if _, err := record.WriteTo(&c.wbuf); err != nil {
		return err
	}
	return nil
}

func (c *ObfsTLSConn) Read(b []byte) (n int, err error) {
	if c.isServer { // NOTE: only Write performs the handshake operation on client side.
		if err = c.Handshake(nil); err != nil {
			return
		}
	}

	<-c.handshaked

	if c.isServer {
		if c.rbuf.Len() > 0 {
			return c.rbuf.Read(b)
		}
		record := &dissector.Record{}
		if _, err = record.ReadFrom(c.Conn); err != nil {
			return
		}
		n = copy(b, record.Opaque)
		_, err = c.rbuf.Write(record.Opaque[n:])
	} else {
		n, err = c.Conn.Read(b)
		if err != nil {
			return
		}
		if n > 0 {
			n, err = c.parser.Parse(b[:n])
		}
	}
	return
}

func (c *ObfsTLSConn) Write(b []byte) (n int, err error) {
	n = len(b)
	if !c.Handshaked() {
		if err = c.Handshake(b); err != nil {
			return
		}
		if !c.isServer { // the data b has been sended during handshake phase.
			return
		}
	}

	for len(b) > 0 {
		data := b
		if len(b) > maxTLSDataLen {
			data = b[:maxTLSDataLen]
			b = b[maxTLSDataLen:]
		} else {
			b = b[:0]
		}
		record := &dissector.Record{
			Type:    dissector.AppData,
			Version: tls.VersionTLS12,
			Opaque:  data,
		}

		if c.wbuf.Len() > 0 {
			record.Type = dissector.Handshake
			record.WriteTo(&c.wbuf)
			_, err = c.wbuf.WriteTo(c.Conn)
			return
		}

		if _, err = record.WriteTo(c.Conn); err != nil {
			return
		}
	}
	return
}
