package transport

import (
	"context"
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/connection/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

type grpcDialer struct {
	tcpDialer
	opts *GrpcOptions
}

func (d *grpcDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	var dialOpts []grpc.DialOption
	var callOpts []grpc.CallOption

	if d.opts.SndBuffer > 0 {
		dialOpts = append(dialOpts, grpc.WithWriteBufferSize(d.opts.SndBuffer))
	}
	if d.opts.RevBuffer > 0 {
		dialOpts = append(dialOpts, grpc.WithReadBufferSize(d.opts.RevBuffer))
	}
	callOpts = append(callOpts, grpc.UseCompressor(gzip.Name))

	cred := insecure.NewCredentials()
	tlsConfig, err := d.opts.TlsOptions.GetClientTlsConfig()
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		cred = credentials.NewTLS(tlsConfig)
	}

	dialOpts = append(dialOpts,
		grpc.WithDefaultCallOptions(callOpts...),
		grpc.WithTransportCredentials(cred),
		grpc.WithContextDialer(d.tcpDialer.Dial),
	)

	conn, err := grpc.DialContext(ctx, addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	client := proto.NewStreamServiceClient(conn)
	cStream, err := client.Transfer(ctx)
	if err != nil {
		return nil, err
	}
	return connection.NewGrpcClientStreamConn(cStream, conn), nil
}
