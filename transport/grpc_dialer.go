package transport

import (
	"context"
	"net"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/connection/proto"
	"github.com/josexy/mini-ss/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
)

type grpcDialer struct {
	tcpDialer
	Opts *GrpcOptions
}

func (d *grpcDialer) Dial(addr string) (net.Conn, error) {
	ctx := context.Background()

	var dialOpts []grpc.DialOption
	var callOpts []grpc.CallOption

	if d.Opts.SndBuffer > 0 {
		dialOpts = append(dialOpts, grpc.WithWriteBufferSize(d.Opts.SndBuffer))
	}
	if d.Opts.RevBuffer > 0 {
		dialOpts = append(dialOpts, grpc.WithReadBufferSize(d.Opts.RevBuffer))
	}
	callOpts = append(callOpts, grpc.UseCompressor(gzip.Name))

	cred := insecure.NewCredentials()
	if d.Opts.TLS {
		var err error
		cred, err = util.LoadClientMTLSCertificate(d.Opts.CertPath, d.Opts.KeyPath, d.Opts.CAPath, d.Opts.Hostname)
		if err != nil {
			return nil, err
		}
	}

	dialOpts = append(dialOpts,
		grpc.WithDefaultCallOptions(callOpts...),
		grpc.WithTransportCredentials(cred),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return d.tcpDialer.Dial(s)
		}),
	)

	conn, err := grpc.DialContext(ctx,
		addr,
		dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	client := proto.NewStreamServiceClient(conn)
	sc, err := client.Transfer(ctx)
	if err != nil {
		return nil, err
	}
	return connection.NewGrpcClientStreamConn(sc, conn), nil
}
