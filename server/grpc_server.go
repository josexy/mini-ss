package server

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/connection/proto"
	"github.com/josexy/mini-ss/transport"
	"github.com/josexy/mini-ss/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "google.golang.org/grpc/encoding/gzip"
)

type GrpcServer struct {
	proto.UnimplementedStreamServiceServer
	ln       *tcpKeepAliveListener
	server   *grpc.Server
	Addr     string
	Handler  GrpcHandler
	mu       sync.Mutex
	closed   uint32
	doneChan chan struct{}
	opts     *transport.GrpcOptions
	err      chan error
}

func NewGrpcServer(addr string, handler GrpcHandler, opts transport.Options) *GrpcServer {
	return &GrpcServer{
		Addr:     addr,
		Handler:  handler,
		doneChan: make(chan struct{}),
		err:      make(chan error, 1),
		opts:     opts.(*transport.GrpcOptions),
		closed:   1,
	}
}

func (s *GrpcServer) Start() {
	laddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		s.err <- err
		return
	}
	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		s.err <- err
		return
	}
	s.ln = &tcpKeepAliveListener{ln}

	var opts []grpc.ServerOption

	if s.opts.SndBuffer > 0 {
		opts = append(opts, grpc.WriteBufferSize(s.opts.SndBuffer))
	}
	if s.opts.RevBuffer > 0 {
		opts = append(opts, grpc.ReadBufferSize(s.opts.RevBuffer))
	}
	cred := insecure.NewCredentials()
	if s.opts.TLS {
		var err error
		cred, err = util.LoadServerMTLSCertificate(s.opts.CertPath, s.opts.KeyPath, s.opts.CAPath, s.opts.Hostname)
		if err != nil {
			s.err <- err
			return
		}
	}
	opts = append(opts, grpc.Creds(cred))

	s.server = grpc.NewServer(opts...)
	proto.RegisterStreamServiceServer(s.server, s)

	s.err <- nil
	atomic.StoreUint32(&s.closed, 0)
	defer s.Close()
	s.server.Serve(s.ln)
}

func (s *GrpcServer) Transfer(ss proto.StreamService_TransferServer) error {
	conn := connection.NewGrpcServerStreamConn(ss, s.ln.Addr())
	if s.Handler != nil {
		s.Handler.ServeGRPC(conn)
	}
	return nil
}

func (s *GrpcServer) Error() chan error { return s.err }

func (s *GrpcServer) Build() Server { return s }

func (s *GrpcServer) LocalAddr() string { return s.Addr }

func (s *GrpcServer) Type() ServerType { return Grpc }

func (s *GrpcServer) Close() error {
	if atomic.LoadUint32(&s.closed) != 0 {
		return ErrServerClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	atomic.StoreUint32(&s.closed, 1)
	close(s.doneChan)
	s.ln.Close()
	s.server.GracefulStop()
	return nil
}

func (s *GrpcServer) Serve(*Conn) {}
