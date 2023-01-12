package bufferpool

import (
	"bytes"
	"sync"
	"testing"
)

const bufferSize = 1024

var testString = bytes.Repeat([]byte{0xAA, 0xBB, 0xCC, 0xDD}, 2)

var bpSlice = sync.Pool{
	New: func() any {
		buf := make([]byte, bufferSize)
		return buf
	},
}

var bpSlicePtr = NewBufferPool(bufferSize)

var bpByteBuffer = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, bufferSize))
	},
}

func BenchmarkWithoutBufferPoolByteBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, bufferSize))
		buf.Reset()
		buf.Write(testString)
	}
}

func BenchmarkWithoutBufferPoolSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := make([]byte, bufferSize)
		copy(buf, testString)
	}
}

func BenchmarkWithBufferPoolByteBuffer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bpByteBuffer.Get()
		buf.(*bytes.Buffer).Write(testString)
		buf.(*bytes.Buffer).Reset()
		bpByteBuffer.Put(buf)
	}
}

func BenchmarkWithBufferPoolSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bpSlice.Get()
		copy(buf.([]byte), testString)
		bpSlice.Put(buf)
	}
}

func BenchmarkWithBufferPoolSlicePtr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bpSlicePtr.Get()
		copy(*buf, testString)
		*buf = (*buf)[:0]
		bpSlicePtr.Put((buf))
	}
}
