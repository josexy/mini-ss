package relay

import (
	"context"
	"io"
	"net"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
)

var (
	tcpPool    = bufferpool.NewBufferPool(bufferpool.MaxTcpBufferSize)
	addrPool   = bufferpool.NewBufferPool(bufferpool.MaxAddressBufferSize)
	emptyBytes = make([]byte, 1)
)

func IoCopyBidirectionalForStream(dst, src io.ReadWriteCloser) error {
	defer dst.Close()
	defer src.Close()
	errCh := make(chan error, 2)
	copyFn := func(dest, src io.ReadWriteCloser) {
		err := ioCopyWithBuffer(dest, src)
		errCh <- err
	}
	go copyFn(dst, src)
	go copyFn(src, dst)
	return <-errCh
}

func ioCopyWithBuffer(dst io.Writer, src io.Reader) error {
	var b []byte
	if _, ok := src.(io.WriterTo); ok {
		b = emptyBytes
	} else if _, ok := dst.(io.ReaderFrom); ok {
		b = emptyBytes
	} else {
		buf := tcpPool.Get()
		defer tcpPool.Put(buf)
		b = *buf
	}
	_, err := io.CopyBuffer(dst, src, b)
	return err
}

type TCPDirectRelayer struct{ transport.Dialer }

func NewTCPDirectRelayer() *TCPDirectRelayer {
	return &TCPDirectRelayer{Dialer: transport.NewDialer(transport.Tcp, nil)}
}

func (r *TCPDirectRelayer) RelayToServer(conn net.Conn, remoteServerAddr string) error {
	dstConn, err := r.Dial(context.Background(), remoteServerAddr)
	if err != nil {
		return err
	}

	logger.Logger.Info("tcp-direct",
		logx.String("relayer", conn.RemoteAddr().String()),
		logx.String("remote", remoteServerAddr),
	)

	return IoCopyBidirectionalForStream(dstConn, conn)
}

type ProxyTCPRelayer struct {
	transport.Dialer
	typ             transport.Type
	inbound         transport.TcpConnBound
	outbound        transport.TcpConnBound
	proxyServerAddr string
}

func NewProxyTCPRelayer(proxyServerAddr string, typ transport.Type, opts options.Options,
	inbound, outbound transport.TcpConnBound) *ProxyTCPRelayer {
	return &ProxyTCPRelayer{
		typ:             typ,
		inbound:         inbound,
		outbound:        outbound,
		Dialer:          transport.NewDialer(typ, opts),
		proxyServerAddr: proxyServerAddr,
	}
}

func (r *ProxyTCPRelayer) RelayToProxyServer(conn net.Conn, remoteServerAddr string) error {
	dstConn, err := r.Dial(context.Background(), r.proxyServerAddr)
	if err != nil {
		return err
	}
	if r.outbound != nil {
		dstConn = r.outbound.TcpConn(dstConn)
	}
	buf := addrPool.Get()
	addr, err := address.ParseAddress(remoteServerAddr, *buf)
	if err != nil {
		addrPool.Put(buf)
		return err
	}
	dstConn.Write(addr)
	addrPool.Put(buf)

	logger.Logger.Info("tcp-relay",
		logx.String("type", r.typ.String()),
		logx.String("client", conn.RemoteAddr().String()),
		logx.String("relayer", conn.LocalAddr().String()),
		logx.String("server", r.proxyServerAddr),
		logx.String("remote", remoteServerAddr),
	)

	return IoCopyBidirectionalForStream(dstConn, conn)
}

func (r *ProxyTCPRelayer) RelayToServer(conn net.Conn) error {
	if r.inbound != nil {
		conn = r.inbound.TcpConn(conn)
	}
	buf := addrPool.Get()
	addr, err := address.ParseAddressFromReader(conn, *buf)
	if err != nil {
		addrPool.Put(buf)
		return err
	}

	remoteAddr := addr.String()
	addrPool.Put(buf)

	dstConn, err := r.Dial(context.Background(), remoteAddr)
	if err != nil {
		return err
	}

	logger.Logger.Info("tcp-relay",
		logx.String("type", r.typ.String()),
		logx.String("client", conn.RemoteAddr().String()),
		logx.String("relayer", conn.LocalAddr().String()),
		logx.String("remote", remoteAddr),
	)

	return IoCopyBidirectionalForStream(dstConn, conn)
}
