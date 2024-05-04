package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDnsResolver(t *testing.T) {
	DefaultResolver = NewDnsResolver([]string{
		"1.1.1.1",
		"2.1.1.1",
		"3.1.1.1",
		"8.8.8.8",
	})
	ips, err := DefaultResolver.LookupIP(context.Background(), "www.example.com")
	assert.Nil(t, err)
	t.Log(ips)
	time.Sleep(time.Second * 6)
}
