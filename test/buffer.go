package test

import (
	"bytes"
	"sync"
)

// Buffer is a simple thread-safe buffer implementing io.Writer
type Buffer struct {
	mux sync.Mutex
	buf bytes.Buffer
}

// Write adds data to the buffer.
func (b *Buffer) Write(data []byte) (int, error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.Write(data)
}

// String returns the buffer as a string.
func (b *Buffer) String() string {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.String()
}

// Bytes returns buffer contents as a slice.
func (b *Buffer) Bytes() []byte {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.Bytes()
}
