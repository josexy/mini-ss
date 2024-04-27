package cert

import (
	"container/list"
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"math/rand"
	"sync"
	"time"
)

var (
	errNoPriKey = errors.New("no private key available")
	errNoCert   = errors.New("no certificate available")
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

type certInfo struct {
	host      string
	expiredAt time.Time
	cert      tls.Certificate
}

type CertPool struct {
	list              *list.List
	lock              sync.RWMutex
	certs             map[string]*list.Element
	maxCapacity       int
	checkInterval     time.Duration
	certExpiredSecond time.Duration
	doneCh            chan struct{}
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
	pool := &CertPool{
		list:              list.New(),
		certs:             make(map[string]*list.Element, 16),
		maxCapacity:       maxCapacity,
		checkInterval:     checkInterval,
		certExpiredSecond: certExpiredSecond,
		doneCh:            make(chan struct{}, 1),
	}
	go pool.check()
	return pool
}

func (p *CertPool) Stop() {
	p.doneCh <- struct{}{}
}

func (p *CertPool) Get(host string) (tls.Certificate, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if ele, ok := p.certs[host]; ok {
		p.list.MoveToBack(ele)
		certInfo := ele.Value.(*certInfo)
		certInfo.expiredAt = time.Now().Add(p.certExpiredSecond)
		return certInfo.cert, nil
	}
	return tls.Certificate{}, errNoCert
}

func (p *CertPool) Add(host string, cert tls.Certificate) {
	p.lock.RLock()
	if ele, ok := p.certs[host]; ok {
		p.lock.RUnlock()
		p.list.MoveToBack(ele)
		ele.Value.(*certInfo).expiredAt = time.Now().Add(p.certExpiredSecond)
		ele.Value.(*certInfo).cert = cert
		return
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()
	newCertInfo := &certInfo{
		host:      host,
		cert:      cert,
		expiredAt: time.Now().Add(p.certExpiredSecond),
	}
	p.certs[host] = p.list.PushBack(newCertInfo)
	if p.list.Len() > p.maxCapacity {
		p.remove(p.list.Front())
	}
}

func (p *CertPool) removeUnitl() {
	p.lock.Lock()
	defer p.lock.Unlock()
	for {
		if p.list.Len() > p.maxCapacity {
			p.remove(p.list.Front())
		}
	}
}

func (p *CertPool) remove(ele *list.Element) {
	p.list.Remove(ele)
	delete(p.certs, ele.Value.(*certInfo).host)
}

func (c *CertPool) check() {
	timer := time.NewTimer(c.checkInterval)
	defer timer.Stop()
	for {
		select {
		case now := <-timer.C:
			c.cleanupExpiredCerts(now)
			timer.Reset(c.checkInterval)
		case <-c.doneCh:
			return
		}
	}
}

func (c *CertPool) cleanupExpiredCerts(now time.Time) {
	var list []*list.Element
	c.lock.RLock()
	for _, cert := range c.certs {
		list = append(list, cert)
	}
	c.lock.RUnlock()

	for _, ele := range list {
		certInfo := ele.Value.(*certInfo)
		if certInfo.expiredAt.Before(now) {
			c.lock.Lock()
			c.remove(ele)
			c.lock.Unlock()
		}
	}
}
