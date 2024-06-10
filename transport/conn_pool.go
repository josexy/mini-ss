package transport

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	roundrobin uint8 = iota
	random
)

type dialFunc[T any] func(ctx context.Context, addr string, inx int) (T, error)

type connSlot[T any] struct {
	conn T
	has  bool
}

var rdIdx = rand.New(rand.NewSource(time.Now().UnixNano()))

type connPool[T any] struct {
	index     int
	size      int
	policy    uint8
	locker    sync.Mutex
	connGroup singleflight.Group
	conns     []*connSlot[T]
}

func newConnPool[T any](size int) *connPool[T] {
	if size <= 0 {
		size = 3
	}
	return &connPool[T]{
		size:   size,
		policy: roundrobin,
		conns:  make([]*connSlot[T], size),
	}
}

func (p *connPool[T]) selIndex(index int) {
	if p.policy == roundrobin {
		p.index = (index + 1) % p.size
		return
	}
	p.index = rdIdx.Intn(p.size)
}

func (p *connPool[T]) close(index int, closeFn func(T) error) {
	if index < 0 || index >= p.size {
		return
	}
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.conns[index] == nil {
		return
	}
	p.conns[index].has = false
	closeFn(p.conns[index].conn)
}

func (p *connPool[T]) getConnWithIndex(ctx context.Context, addr string, index int, reselect bool, dialFn dialFunc[T]) (conn T, err error) {
	if index < 0 || index >= p.size {
		return
	}
	newCtx, newCancel := context.WithCancel(ctx)
	ch := p.connGroup.DoChan(strconv.FormatInt(int64(index), 10), func() (interface{}, error) {
		conn, err := dialFn(newCtx, addr, index)
		if err != nil {
			return nil, err
		}
		p.locker.Lock()
		if p.conns[index] == nil {
			p.conns[index] = new(connSlot[T])
		}
		p.conns[index].conn = conn
		p.conns[index].has = true
		if reselect {
			p.selIndex(index)
		}
		p.locker.Unlock()
		return conn, err
	})
	select {
	case <-ctx.Done():
		newCancel()
		err = ctx.Err()
		return
	case r := <-ch:
		newCancel()
		if r.Err != nil {
			err = r.Err
			return
		}
		conn, _ = r.Val.(T)
		return
	}
}

func (p *connPool[T]) getConn(ctx context.Context, addr string, dialFn dialFunc[T]) (conn T, err error) {
	p.locker.Lock()
	index := p.index
	if p.conns[index] != nil && p.conns[index].has {
		conn = p.conns[index].conn
		p.selIndex(index)
		p.locker.Unlock()
		return
	}
	p.locker.Unlock()

	return p.getConnWithIndex(ctx, addr, index, true, dialFn)
}
