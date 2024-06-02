package server

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"sync/atomic"

	"github.com/josexy/mini-ss/connection"
	"github.com/josexy/mini-ss/options"
	"golang.org/x/crypto/ssh"
)

var (
	errPasswordAuthFailed  = errors.New("ssh password authentication failed")
	errPublicKeyAuthFailed = errors.New("ssh public key authentication failed")
)

var _ Server = (*SshServer)(nil)

type SshServer struct {
	ln      *tcpKeepAliveListener
	Addr    string
	Handler SshHandler
	opts    *options.SshOptions
	running atomic.Bool
}

func NewSshServer(addr string, handler SshHandler, opts options.Options) *SshServer {
	return &SshServer{
		Addr:    addr,
		Handler: handler,
		opts:    opts.(*options.SshOptions),
	}
}

func (s *SshServer) Start(ctx context.Context) error {
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

	sshConfig, err := s.newSshConfig()
	if err != nil {
		return err
	}

	s.running.Store(true)
	go closeWithContextDoneErr(ctx, s)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if !s.running.Load() {
				break
			}
			continue
		}
		sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
		if err != nil {
			continue
		}
		go func(reqs <-chan *ssh.Request) {
			for req := range reqs {
				if req.Type == "keepalive@openssh.com" {
					req.Reply(true, nil)
				}
			}
		}(reqs)
		go func(chs <-chan ssh.NewChannel, conn *ssh.ServerConn) {
			for ch := range chs {
				go s.handlePerChannel(ch, conn)
			}
		}(chans, sshConn)
	}
	return err
}

func (s *SshServer) handlePerChannel(newCh ssh.NewChannel, conn *ssh.ServerConn) {
	chType := newCh.ChannelType()
	if chType != "direct-tcpip" {
		newCh.Reject(ssh.UnknownChannelType, "unknown channel type: "+chType)
		return
	}
	channel, reqs, err := newCh.Accept()
	if err != nil {
		return
	}
	go ssh.DiscardRequests(reqs)
	newConn(connection.NewSshConn(channel, conn.LocalAddr(), conn.RemoteAddr()), s).serve()
}

func (s *SshServer) newSshConfig() (*ssh.ServerConfig, error) {
	sshConfig := &ssh.ServerConfig{}

	privateKeyData, err := os.ReadFile(s.opts.PrivateKey)
	if err != nil {
		return nil, err
	}
	privateKey, err := ssh.ParsePrivateKey(privateKeyData)
	if err != nil {
		return nil, err
	}
	sshConfig.AddHostKey(privateKey)

	switch {
	case s.opts.Password != "":
		sshConfig.PasswordCallback = func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if s.opts.User == conn.User() && s.opts.Password == string(password) {
				return nil, nil
			}
			return nil, errPasswordAuthFailed
		}
	case s.opts.AuthorizedKey != "":
		keys, err := getPublicKeysFromAuthorizedKeyFile(s.opts.AuthorizedKey)
		if err != nil {
			return nil, err
		}
		sshConfig.PublicKeyCallback = func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if s.opts.User != conn.User() {
				return nil, errPublicKeyAuthFailed
			}
			publicKeyData := key.Marshal()
			authOk := false
			for _, key := range keys {
				if bytes.Compare(publicKeyData, key) == 0 {
					authOk = true
					break
				}
			}
			if !authOk {
				return nil, errPublicKeyAuthFailed
			}
			return nil, nil
		}
	default:
		sshConfig.NoClientAuth = true
	}
	return sshConfig, nil
}

func getPublicKeysFromAuthorizedKeyFile(file string) ([][]byte, error) {
	var keys [][]byte
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	for len(data) > 0 {
		key, _, _, rest, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			continue
		}
		keys = append(keys, key.Marshal())
		data = rest
	}
	return keys, nil
}

func (s *SshServer) LocalAddr() string { return s.Addr }

func (s *SshServer) Type() ServerType { return Ssh }

func (s *SshServer) Close() error {
	if !s.running.Load() {
		return ErrServerClosed
	}
	s.running.Store(false)
	return s.ln.Close()
}

func (s *SshServer) Serve(conn *Conn) {
	if s.Handler != nil {
		s.Handler.ServeSSH(conn)
	}
}
