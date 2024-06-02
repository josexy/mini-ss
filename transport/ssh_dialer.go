package transport

import (
	"context"
	"net"
	"os"
	"sync"
	"time"

	"github.com/josexy/mini-ss/options"
	"golang.org/x/crypto/ssh"
)

type sshDialer struct {
	tcpDialer
	opts       *options.SshOptions
	signerOnce sync.Once
	once       sync.Once
	signer     ssh.Signer
	signerErr  error
	err        error
	client     *ssh.Client
}

func (d *sshDialer) initClient(ctx context.Context, addr string) (*ssh.Client, error) {
	d.signerOnce.Do(func() {
		if d.opts.PublicKey != "" {
			var privateKey []byte
			privateKey, d.signerErr = os.ReadFile(d.opts.PrivateKey)
			if d.signerErr != nil {
				return
			}
			d.signer, d.signerErr = ssh.ParsePrivateKey(privateKey)
		}
	})

	var authMethod []ssh.AuthMethod
	if d.opts.Password != "" {
		authMethod = append(authMethod, ssh.Password(d.opts.Password))
	}
	if d.opts.PublicKey != "" {
		if d.signerErr != nil {
			return nil, d.err
		}
		authMethod = append(authMethod, ssh.PublicKeys(d.signer))
	}
	sshConfig := &ssh.ClientConfig{
		User:            d.opts.User,
		Auth:            authMethod,
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	conn, err := d.tcpDialer.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, err
	}
	d.client = ssh.NewClient(c, chans, reqs)
	return d.client, nil
}

func (d *sshDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	d.once.Do(func() {
		_, err := d.initClient(ctx, addr)
		d.err = err
	})
	if d.err != nil {
		return nil, d.err
	}
	return d.client.DialContext(ctx, "tcp", d.client.RemoteAddr().String())
}
