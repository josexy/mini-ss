package transport

import (
	"context"
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/connection/proto"
	"github.com/josexy/mini-ss/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

type grpcDialer struct {
	tcpDialer
	err      error
	dialOpts []grpc.DialOption
	opts     *options.GrpcOptions
}

func newGRPCDialer(opt options.Options) *grpcDialer {
	opt.Update()
	grpcOpts := opt.(*options.GrpcOptions)
	var dialOpts []grpc.DialOption
	var callOpts []grpc.CallOption

	if grpcOpts.SndBuffer > 0 {
		dialOpts = append(dialOpts, grpc.WithWriteBufferSize(grpcOpts.SndBuffer))
	}
	if grpcOpts.RevBuffer > 0 {
		dialOpts = append(dialOpts, grpc.WithReadBufferSize(grpcOpts.RevBuffer))
	}
	callOpts = append(callOpts, grpc.UseCompressor(gzip.Name))

	cred := insecure.NewCredentials()
	tlsConfig, err := grpcOpts.TlsOptions.GetClientTlsConfig()
	if tlsConfig != nil {
		cred = credentials.NewTLS(tlsConfig)
	}
	grpcDialer := &grpcDialer{
		err:  err,
		opts: grpcOpts,
	}
	dialOpts = append(dialOpts,
		grpc.WithDefaultCallOptions(callOpts...),
		grpc.WithTransportCredentials(cred),
		grpc.WithContextDialer(grpcDialer.tcpDialer.Dial),
	)
	grpcDialer.dialOpts = dialOpts
	return grpcDialer
}

func (d *grpcDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	if d.err != nil {
		return nil, d.err
	}
	conn, err := grpc.DialContext(ctx, addr, d.dialOpts...)
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
