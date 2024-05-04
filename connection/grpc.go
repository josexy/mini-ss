package connection

import (
	"net"
	"time"

	"github.com/josexy/mini-ss/connection/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

/*
Generate golang code from proto buffer file:
> protoc --proto_path=./proto \
	--go_out=paths=source_relative:./proto \
	--go-grpc_out=paths=source_relative:./proto ./proto/stream.proto
*/

type grpcStream interface {
	SendMsg(interface{}) error
	RecvMsg(interface{}) error
}

type grpcStreamReaderWriter struct {
	grpcStream
	reqMsg *proto.PacketData
	rspMsg *proto.PacketData
	rbuf   []byte // remaining buffer data
}

func newGrpcStreamReaderWriter(stream grpcStream) *grpcStreamReaderWriter {
	return &grpcStreamReaderWriter{
		grpcStream: stream,
		reqMsg:     new(proto.PacketData),
		rspMsg:     new(proto.PacketData),
	}
}

func (rw *grpcStreamReaderWriter) Read(b []byte) (int, error) {
	if len(rw.rbuf) == 0 {
		rw.rspMsg.Reset()
		err := rw.RecvMsg(rw.rspMsg)
		if err != nil {
			return 0, err
		}
		rw.rbuf = rw.rspMsg.Data
	}
	n := copy(b, rw.rbuf)
	rw.rbuf = rw.rbuf[n:]
	return n, nil
}

func (rw *grpcStreamReaderWriter) Write(b []byte) (int, error) {
	rw.reqMsg.Reset()
	rw.reqMsg.Data = b
	return len(b), rw.SendMsg(rw.reqMsg)
}

type GrpcStreamConn struct {
	*grpcStreamReaderWriter
	sStream    grpc.ServerStream
	cStream    grpc.ClientStream
	clientConn *grpc.ClientConn
	localAddr  net.Addr
	remoteAddr net.Addr
	isServer   bool
}

func NewGrpcServerStreamConn(sStream grpc.ServerStream) *GrpcStreamConn {
	var lAddr, rAddr net.Addr
	if peerCtx, ok := peer.FromContext(sStream.Context()); ok {
		lAddr = peerCtx.LocalAddr
		rAddr = peerCtx.Addr
	}
	return &GrpcStreamConn{
		sStream:                sStream,
		localAddr:              lAddr,
		remoteAddr:             rAddr,
		isServer:               true,
		grpcStreamReaderWriter: newGrpcStreamReaderWriter(sStream),
	}
}

func NewGrpcClientStreamConn(cStream grpc.ClientStream, conn *grpc.ClientConn) *GrpcStreamConn {
	var lAddr, rAddr net.Addr
	if peerCtx, ok := peer.FromContext(cStream.Context()); ok {
		lAddr = peerCtx.LocalAddr
		rAddr = peerCtx.Addr
	}
	return &GrpcStreamConn{
		cStream:                cStream,
		clientConn:             conn,
		localAddr:              lAddr,
		remoteAddr:             rAddr,
		grpcStreamReaderWriter: newGrpcStreamReaderWriter(cStream),
	}
}

func (c *GrpcStreamConn) Close() error {
	if !c.isServer {
		_ = c.cStream.CloseSend()
		return c.clientConn.Close()
	}
	return nil
}

func (c *GrpcStreamConn) LocalAddr() net.Addr { return c.localAddr }

func (c *GrpcStreamConn) RemoteAddr() net.Addr { return c.remoteAddr }

func (c *GrpcStreamConn) SetReadDeadline(t time.Time) error { return nil }

func (c *GrpcStreamConn) SetWriteDeadline(t time.Time) error { return nil }

func (c *GrpcStreamConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}
