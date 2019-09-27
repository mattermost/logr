package test

import (
	"bytes"
	"math/rand"
	"sync"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// StringRnd returns a pseudo-random string of the specified length.
func StringRnd(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Buffer is a simple buffer implementing io.Writer
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
