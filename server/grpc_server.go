package server

import (
	"context"
	"errors"
	"net"
	"sync/atomic"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/connection/proto"
	"github.com/josexy/mini-ss/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip"
)

var _ Server = (*GrpcServer)(nil)

type GrpcServer struct {
	proto.UnimplementedStreamServiceServer
	ln      *tcpKeepAliveListener
	server  *grpc.Server
	Addr    string
	Handler GrpcHandler
	opts    *options.GrpcOptions
	running atomic.Bool
}

func NewGrpcServer(addr string, handler GrpcHandler, opts options.Options) *GrpcServer {
	return &GrpcServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*options.GrpcOptions),
	}
}

func (s *GrpcServer) Start(ctx context.Context) error {
	if s.running.Load() {
		return ErrServerStarted
	}
	laddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		return err
	}
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}
	s.ln = &tcpKeepAliveListener{ln}

	var opts []grpc.ServerOption
	cred := insecure.NewCredentials()
	tlsConfig, err := s.opts.TlsOptions.GetServerTlsConfig()
	if err != nil {
		return err
	}
	if tlsConfig != nil {
		cred = credentials.NewTLS(tlsConfig)
	}
	opts = append(opts, grpc.Creds(cred))
	if s.opts.SndBuffer > 0 {
		opts = append(opts, grpc.WriteBufferSize(s.opts.SndBuffer))
	}
	if s.opts.RevBuffer > 0 {
		opts = append(opts, grpc.ReadBufferSize(s.opts.RevBuffer))
	}

	s.server = grpc.NewServer(opts...)
	proto.RegisterStreamServiceServer(s.server, s)

	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	err = s.server.Serve(s.ln)
	if err != nil && errors.Is(err, grpc.ErrServerStopped) {
		err = nil
	}
	s.running.Store(false)
	return err
}

func (s *GrpcServer) Transfer(ss proto.StreamService_TransferServer) error {
	newConn(connection.NewGrpcServerStreamConn(ss), s).serve()
	return nil
}

func (s *GrpcServer) LocalAddr() string { return s.Addr }

func (s *GrpcServer) Type() ServerType { return Grpc }

func (s *GrpcServer) Close() error {
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	s.server.GracefulStop()
	return nil
}

func (s *GrpcServer) Serve(conn *Conn) {
	if s.Handler != nil {
		s.Handler.ServeGRPC(conn)
	}
}
