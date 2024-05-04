package cache

import (
	"crypto/tls"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheExpired(t *testing.T) {
	cache := NewCache[string, int](
		WithExpiration(time.Second*2),
		WithInterval(time.Second),
		WithUpdateCacheExpirationOnGet(),
		WithEvictCallback(func(k, v any) { t.Logf("evict: %v", k) }),
	)

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 4)
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			cache.Set("key"+strconv.Itoa(i), i)
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 1)
		for i := 4; i <= 7; i++ {
			cache.Get("key" + strconv.Itoa(i))
		}
	}()

	go func() {
		defer wg.Done()
		time.Sleep(time.Second * 6)
		for i := 4; i <= 7; i++ {
			_, err := cache.Get("key" + strconv.Itoa(i))
			t.Logf("key_%d, err: %v", i, err)
			assert.Equal(t, err, ErrCacheWasExpired)
		}
	}()

	wg.Wait()

	cache.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestCacheBackgroundCheckExpired(t *testing.T) {
	cache := NewCache[string, tls.Certificate](
		WithMaxSize(10),
		WithInterval(time.Second*3),
		WithExpiration(time.Second*2),
		WithBackgroundCheckCache(),
		WithUpdateCacheExpirationOnGet(),
		WithEvictCallback(func(k, v any) { t.Logf("evict: %v", k) }),
	)
	for i := 0; i < 15; i++ {
		key := "key_" + strconv.Itoa(i)
		cache.Set(key, tls.Certificate{})
		t.Log(key, "Added")
	}

	go func() {
		time.Sleep(time.Second * 2)
		for i := 8; i <= 13; i++ {
			_, err := cache.Get("key_" + strconv.Itoa(i))
			assert.Nil(t, err)
		}
	}()

	go func() {
		time.Sleep(time.Millisecond * 6700)
		for i := 0; i < 15; i++ {
			_, err := cache.Get("key_" + strconv.Itoa(i))
			t.Logf("key_%d, err: %v", i, err)
			assert.Equal(t, err, ErrCacheIsNotFound)
		}
	}()
	time.Sleep(time.Second * 7)
}

func BenchmarkCacheSetAndGet(b *testing.B) {
	cache := NewCache[string, int](
		WithMaxSize(100),
		WithExpiration(time.Second*2),
		WithInterval(time.Millisecond*10),
		WithBackgroundCheckCache(),
	)
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			key := "key" + strconv.Itoa(rand.Intn(40))
			cache.Set(key, 1)
			key = "key" + strconv.Itoa(rand.Intn(40))
			cache.Get(key)
		}
	})
}
