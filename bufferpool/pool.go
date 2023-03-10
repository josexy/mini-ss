package bufferpool

import (
	"bytes"
	"sync"
)

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool(size int) *BufferPool {
	bp := new(BufferPool)
	bp.pool.New = func() any {
		buf := make([]byte, size)
		return &buf
	}
	return bp
}

func NewBytesBufferPool() *BufferPool {
	bp := new(BufferPool)
	bp.pool.New = func() any {
		return &bytes.Buffer{}
	}
	return bp
}

func (bp *BufferPool) Get() *[]byte {
	return bp.pool.Get().(*[]byte)
}

func (bp *BufferPool) Put(buf *[]byte) {
	bp.pool.Put(buf)
}

func (bp *BufferPool) GetBytesBuffer() *bytes.Buffer {
	return bp.pool.Get().(*bytes.Buffer)
}

func (bp *BufferPool) PutBytesBuffer(buf *bytes.Buffer) {
	// reset buffer
	buf.Reset()
	bp.pool.Put(buf)
}
