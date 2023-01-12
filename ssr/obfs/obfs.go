package obfs

import (
	"errors"
	"fmt"
	"net"
)

var (
	errTLS12TicketAuthIncorrectMagicNumber = errors.New("tls1.2_ticket_auth incorrect magic number")
	errTLS12TicketAuthTooShortData         = errors.New("tls1.2_ticket_auth too short data")
	errTLS12TicketAuthHMACError            = errors.New("tls1.2_ticket_auth hmac verifying failed")
)

const relayBufferSize = 20 * 1024

const (
	obfsPlain               = "plain"
	obfsHttpSimple          = "http_simple"
	obfsHttpPost            = "http_post"
	obfsRandomHead          = "random_head"
	obfsTLS12TicketAuth     = "tls1.2_ticket_auth"
	obfsTLS12TicketFastAuth = "tls1.2_ticket_fastauth"
)

type authData struct {
	clientID [32]byte
}

// Obfs only support TCP protocol
type Obfs interface {
	StreamConn(net.Conn) net.Conn
}

var obfsMap = map[string]obfsWrapper{
	obfsPlain:               {0, newPlain},
	obfsHttpSimple:          {0, newHTTPSimple},
	obfsHttpPost:            {0, newHTTPPost},
	obfsRandomHead:          {0, newRandomHead},
	obfsTLS12TicketAuth:     {5, newTLS12Ticket},
	obfsTLS12TicketFastAuth: {5, newTLS12Ticket},
}

type obfsWrapper struct {
	Overhead int
	New      func(b *Base) Obfs
}

func GetObfs(name string, b *Base) (Obfs, int, error) {
	if obfs, ok := obfsMap[name]; ok {
		return obfs.New(b), obfs.Overhead, nil
	}
	return nil, 0, fmt.Errorf("Obfs %s not supported", name)
}
