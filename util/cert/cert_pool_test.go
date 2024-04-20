package cert

import (
	"crypto/tls"
	"strconv"
	"testing"
	"time"
)

func TestCertPool(t *testing.T) {
	pool := NewCertPool(10, time.Second*3, time.Second*1)
	for i := 0; i < 15; i++ {
		key := "host_" + strconv.Itoa(i)
		pool.Add(key, tls.Certificate{})
		t.Log(key, "Added")
	}

	go func() {
		time.Sleep(time.Second * 2)
		for i := 8; i < 13; i++ {
			pool.Get("host_" + strconv.Itoa(i))
		}
	}()

	time.Sleep(time.Second * 6)
}
