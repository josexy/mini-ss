package cert

import (
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"math/rand"
	"time"

	"github.com/josexy/mini-ss/util/cache"
)

var (
	errNoPriKey = errors.New("no private key available")
)

type PriKeyPool struct {
	rand *rand.Rand
	keys []*rsa.PrivateKey
}

func NewPriKeyPool(maxSize int) *PriKeyPool {
	if maxSize <= 0 {
		maxSize = 10
	}
	pool := &PriKeyPool{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
		keys: make([]*rsa.PrivateKey, 0, maxSize),
	}
	return pool
}

func (p *PriKeyPool) Get() (*rsa.PrivateKey, error) {
	var n, m = len(p.keys), cap(p.keys)
	if m == 0 {
		return nil, errNoPriKey
	}
	if n < m {
		key, err := GeneratePrivateKey()
		if err != nil {
			return nil, err
		}
		p.keys = append(p.keys, key)
		return key, nil
	}
	index := p.rand.Intn(n)
	key := p.keys[index]
	return key, nil
}

type CertPool struct {
	cache.Cache[string, tls.Certificate]
}

func NewCertPool(maxCapacity int, checkInterval, certExpiredSecond time.Duration) *CertPool {
	if maxCapacity <= 0 {
		maxCapacity = 50
	}
	if checkInterval <= 0 {
		checkInterval = time.Second * 30
	}
	if certExpiredSecond <= 0 {
		certExpiredSecond = time.Second * 15
	}
	return &CertPool{
		Cache: cache.NewCache[string, tls.Certificate](
			cache.WithMaxSize(maxCapacity),
			cache.WithInterval(checkInterval),
			cache.WithExpiration(certExpiredSecond),
			cache.WithBackgroundCheckCache(),
			cache.WithUpdateCacheExpirationOnGet(),
			// cache.WithDeleteExpiredCacheOnGet(),
		),
	}
}
