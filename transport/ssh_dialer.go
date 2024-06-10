package transport

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/josexy/mini-ss/options"
	"github.com/josexy/mini-ss/util/logger"
	"golang.org/x/crypto/ssh"
)

type sshClient struct {
	addr string
	idx  int
	*ssh.Client
}

type sshDialer struct {
	tcpDialer
	err       error
	sshConfig *ssh.ClientConfig
	opts      *options.SshOptions
	cpool     *connPool[*sshClient]
}

func newSSHDialer(opt options.Options) *sshDialer {
	opt.Update()
	sshOpts := opt.(*options.SshOptions)
	var authMethod []ssh.AuthMethod
	if sshOpts.Password != "" {
		authMethod = append(authMethod, ssh.Password(sshOpts.Password))
	}
	var err error
	if sshOpts.PublicKey != "" {
		var privateKey []byte
		if privateKey, err = os.ReadFile(sshOpts.PrivateKey); err == nil {
			var signer ssh.Signer
			if signer, err = ssh.ParsePrivateKey(privateKey); err == nil {
				authMethod = append(authMethod, ssh.PublicKeys(signer))
			}
		}
	}
	return &sshDialer{
		err:   err,
		opts:  sshOpts,
		cpool: newConnPool[*sshClient](3),
		sshConfig: &ssh.ClientConfig{
			User:            sshOpts.User,
			Auth:            authMethod,
			Timeout:         15 * time.Second,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
	}
}

func (d *sshDialer) initClient(ctx context.Context, addr string) (*sshClient, error) {
	conn, err := d.tcpDialer.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, d.sshConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}
	client := &sshClient{
		addr:   addr,
		Client: ssh.NewClient(c, chans, reqs),
	}
	return client, nil
}

func (d *sshDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	if d.opts.PublicKey != "" {
		if d.err != nil {
			return nil, d.err
		}
	}

	client, err := d.getAndDial(ctx, addr)
	if err != nil {
		return nil, err
	}

	return d.dial(ctx, client)
}

func (d *sshDialer) getAndDial(ctx context.Context, addr string) (*sshClient, error) {
	return d.cpool.getConn(ctx, addr, func(ctx context.Context, addr string, idx int) (*sshClient, error) {
		client, err := d.initClient(ctx, addr)
		if err != nil {
			return nil, err
		}
		client.idx = idx
		return client, nil
	})
}

func (d *sshDialer) retryDial(ctx context.Context, addr string, index int) (*sshClient, error) {
	return d.cpool.getConnWithIndex(ctx, addr, index, false, func(ctx context.Context, addr string, idx int) (*sshClient, error) {
		client, err := d.initClient(ctx, addr)
		if err != nil {
			return nil, err
		}
		client.idx = idx
		return client, nil
	})
}

func (d *sshDialer) dial(ctx context.Context, client *sshClient) (net.Conn, error) {
	var err error
	var conn net.Conn
	var fails, retries = 0, 1
	for {
		newCtx, cancel := context.WithTimeout(ctx, time.Second*15)
		if conn, err = client.DialContext(newCtx, "tcp", client.addr); err == nil {
			cancel()
			break
		}
		cancel()

		if fails >= retries {
			return nil, err
		}
		d.cpool.close(client.idx, func(sc *sshClient) error { return sc.Close() })
		newCtx, cancel = context.WithTimeout(ctx, time.Second*15)
		if client, err = d.retryDial(newCtx, client.addr, client.idx); err != nil {
			cancel()
			return nil, err
		}
		cancel()
		fails++
	}
	logger.Logger.Tracef("ssh dial connection: %s, idx:[%d]", client.LocalAddr(), client.idx)
	return conn, nil
}
