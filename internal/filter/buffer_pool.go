package filter

import (
	"bytes"
	"sync"
)

// BufferPool is a zero-copy buffer pool for pipeline stages.
// It reuses *bytes.Buffer allocations to reduce GC pressure when many
// short-lived buffers are created during filter Apply() calls.
//
// Task #154: Zero-copy buffer pool for pipeline stages.
var BufferPool = newBufferPool()

type bufferPool struct {
	pool sync.Pool
}

func newBufferPool() *bufferPool {
	return &bufferPool{
		pool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},
	}
}

// Get returns a ready-to-use buffer with length reset to zero.
func (p *bufferPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool. The buffer is reset before pooling.
// Callers must not use buf after calling Put.
func (p *bufferPool) Put(buf *bytes.Buffer) {
	// Cap at 64 KiB — return large buffers to GC to avoid unbounded memory.
	if buf.Cap() > 64*1024 {
		return
	}
	buf.Reset()
	p.pool.Put(buf)
}

// StringPool is a pool of string builders for building compressed output lines.
var StringPool = newStringPool()

type stringPool struct {
	pool sync.Pool
}

func newStringPool() *stringPool {
	return &stringPool{
		pool: sync.Pool{
			New: func() any {
				var sb bytes.Buffer
				sb.Grow(2048)
				return &sb
			},
		},
	}
}

// Get returns a reset string builder.
func (p *stringPool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns the builder to the pool.
func (p *stringPool) Put(buf *bytes.Buffer) {
	if buf.Cap() > 128*1024 {
		return
	}
	buf.Reset()
	p.pool.Put(buf)
}

// WithBuffer executes fn with a pooled buffer, returning the buffer's string
// content and automatically returning the buffer to the pool.
func WithBuffer(fn func(buf *bytes.Buffer)) string {
	buf := BufferPool.Get()
	fn(buf)
	result := buf.String()
	BufferPool.Put(buf)
	return result
}
