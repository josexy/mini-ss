package socks

import (
	"github.com/josexy/mini-ss/socks/client"
)

func NewSocks5Client(addr string) *client.Socks5Client {
	return client.NewSocks5Client(addr)
}
