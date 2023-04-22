package connection

import (
	"net"
	"time"

	"github.com/josexy/mini-ss/connection/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

/*
generate protobufs code:
protoc --proto_path=./proto \
	--go_out=paths=source_relative:./proto \
	--go-grpc_out=paths=source_relative:./proto ./proto/stream.proto
*/

type ConnExt interface {
	net.Conn
	CloseSender() error
}

type grpcStream interface {
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}

type grpcRW struct {
	grpcRW grpcStream
	req    *proto.PacketData
	resp   *proto.PacketData
	rbuf   []byte // remaining buffer data
}

func newGrpcRW(rw grpcStream) *grpcRW {
	return &grpcRW{
		grpcRW: rw,
		req:    new(proto.PacketData),
		resp:   new(proto.PacketData),
	}
}

func (rw *grpcRW) Read(b []byte) (int, error) {
	if len(rw.rbuf) == 0 {
		rw.resp.Reset()
		err := rw.grpcRW.RecvMsg(rw.resp)
		if err != nil {
			return 0, err
		}
		rw.rbuf = rw.resp.Data
	}
	n := copy(b, rw.rbuf)
	rw.rbuf = rw.rbuf[n:]
	return n, nil
}

func (rw *grpcRW) Write(b []byte) (int, error) {
	rw.req.Reset()
	rw.req.Data = b
	return len(b), rw.grpcRW.SendMsg(rw.req)
}

var _ ConnExt = &GrpcStreamConn{}

type GrpcStreamConn struct {
	*grpcRW
	ss         grpc.ServerStream
	sc         grpc.ClientStream
	clientConn *grpc.ClientConn
	localAddr  net.Addr
	remoteAddr net.Addr
	isServer   bool
}

func NewGrpcServerStreamConn(ss grpc.ServerStream, lAddr net.Addr) *GrpcStreamConn {
	peer, ok := peer.FromContext(ss.Context())
	var rAddr net.Addr
	if ok {
		rAddr = peer.Addr
	}
	return &GrpcStreamConn{
		ss:         ss,
		grpcRW:     newGrpcRW(ss),
		localAddr:  lAddr,
		remoteAddr: rAddr,
		isServer:   true,
	}
}

func NewGrpcClientStreamConn(sc grpc.ClientStream, conn *grpc.ClientConn) *GrpcStreamConn {
	peer, ok := peer.FromContext(sc.Context())
	var rAddr net.Addr
	if ok {
		rAddr = peer.Addr
	}
	lAddr, _ := net.ResolveTCPAddr("tcp", conn.Target())
	return &GrpcStreamConn{
		sc:         sc,
		clientConn: conn,
		grpcRW:     newGrpcRW(sc),
		localAddr:  lAddr,
		remoteAddr: rAddr,
	}
}

func (c *GrpcStreamConn) Close() error {
	if !c.isServer {
		c.sc.CloseSend()
		return c.clientConn.Close()
	}
	return nil
}

func (c *GrpcStreamConn) CloseSender() error {
	if !c.isServer {
		return c.sc.CloseSend()
	}
	return nil
}

func (c *GrpcStreamConn) LocalAddr() net.Addr { return c.localAddr }

func (c *GrpcStreamConn) RemoteAddr() net.Addr { return c.remoteAddr }

func (c *GrpcStreamConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *GrpcStreamConn) SetReadDeadline(t time.Time) error { return nil }

func (c *GrpcStreamConn) SetWriteDeadline(t time.Time) error { return nil }
