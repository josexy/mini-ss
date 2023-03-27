package relay

import (
	"io"
	"net"
	"sync"

	"github.com/josexy/logx"
	"github.com/josexy/mini-ss/address"
	"github.com/josexy/mini-ss/bufferpool"
	"github.com/josexy/mini-ss/constant"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util/logger"
)

var tcpPool = bufferpool.NewBufferPool(constant.MaxTcpBufferSize)

func RelayTCP(dst, src io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)
	fn := func(dest, src io.ReadWriteCloser) {
		defer wg.Done()
		_ = ioCopyWithBuffer(dest, src)
		_ = dest.Close()
	}
	go fn(dst, src)
	go fn(src, dst)
	wg.Wait()
}

func ioCopyWithBuffer(dst io.Writer, src io.Reader) error {
	buf := tcpPool.Get()
	defer tcpPool.Put(buf)
	_, err := io.CopyBuffer(dst, src, *buf)
	return err
}

type TCPDirectRelayer struct{ transport.Dialer }

func NewTCPDirectRelayer() *TCPDirectRelayer {
	return &TCPDirectRelayer{Dialer: transport.NewDialer(transport.Tcp, nil)}
}

type DstTCPRelayer struct {
	DstAddr string
	*TCPRelayer
}

func (r *TCPDirectRelayer) RelayDirectTCP(relayer net.Conn, remoteAddr string) error {
	dstConn, err := r.Dial(remoteAddr)
	if err != nil {
		return err
	}

	logger.Logger.Info("tcp-direct",
		logx.String("relayer", relayer.RemoteAddr().String()),
		logx.String("remote", remoteAddr),
	)

	RelayTCP(dstConn, relayer)
	return nil
}

type TCPRelayer struct {
	transport.Dialer
	typ      transport.Type
	inbound  transport.TcpConnBound
	outbound transport.TcpConnBound
}

func NewTCPRelayer(typ transport.Type, opts transport.Options,
	inbound, outbound transport.TcpConnBound) *TCPRelayer {
	return &TCPRelayer{
		Dialer:   transport.NewDialer(typ, opts),
		typ:      typ,
		inbound:  inbound,
		outbound: outbound,
	}
}

func (r *TCPRelayer) RelayLocalToServer(relayer net.Conn, serverAddr, remoteAddr string) error {
	dstConn, err := r.Dial(serverAddr)
	if err != nil {
		return err
	}
	if r.outbound != nil {
		dstConn = r.outbound.TcpConn(dstConn)
	}
	addr := address.ParseAddress1(remoteAddr)
	dstConn.Write(addr)

	logger.Logger.Info("tcp-relay",
		logx.String("type", r.typ.String()),
		logx.String("client", relayer.RemoteAddr().String()),
		logx.String("relayer", relayer.LocalAddr().String()),
		logx.String("server", serverAddr),
		logx.String("remote", remoteAddr),
	)

	RelayTCP(dstConn, relayer)
	return nil
}

func (r *TCPRelayer) RelayServerToRemote(relayer net.Conn) error {
	if r.inbound != nil {
		relayer = r.inbound.TcpConn(relayer)
	}
	// parse the remote server address
	addr, buf, err := address.ParseAddress4(relayer)
	if err != nil {
		return err
	}

	remoteAddr := addr.String()
	address.PutAddrBuf(buf)

	dstConn, err := r.Dial(remoteAddr)
	if err != nil {
		return err
	}

	logger.Logger.Info("tcp-relay",
		logx.String("type", r.typ.String()),
		logx.String("client", relayer.RemoteAddr().String()),
		logx.String("relayer", relayer.LocalAddr().String()),
		logx.String("remote", remoteAddr),
	)

	RelayTCP(dstConn, relayer)
	return nil
}
