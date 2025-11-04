package logr_test

import (
	"sync"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuffer_ReadWrite(t *testing.T) {
	buf := &logr.Buffer{}

	// Write some data
	data := []byte("test data")
	n, err := buf.Write(data)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	// Read it back
	readBuf := make([]byte, len(data))
	n, err = buf.Read(readBuf)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, readBuf)
}

func TestBuffer_String(t *testing.T) {
	buf := &logr.Buffer{}

	// Write some data
	testStr := "Hello, World!"
	_, err := buf.Write([]byte(testStr))
	require.NoError(t, err)

	// Get string representation
	result := buf.String()
	assert.Equal(t, testStr, result)
}

func TestBuffer_MultipleWrites(t *testing.T) {
	buf := &logr.Buffer{}

	writes := []string{
		"first ",
		"second ",
		"third",
	}

	for _, w := range writes {
		_, err := buf.Write([]byte(w))
		require.NoError(t, err)
	}

	expected := "first second third"
	assert.Equal(t, expected, buf.String())
}

func TestBuffer_ConcurrentWrites(t *testing.T) {
	buf := &logr.Buffer{}
	iterations := 100
	var wg sync.WaitGroup

	// Launch multiple goroutines writing concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = buf.Write([]byte("X"))
			}
		}(i)
	}

	wg.Wait()

	// Should have 10 * iterations X's without panic or race
	result := buf.String()
	assert.Equal(t, 10*iterations, len(result))
	for _, ch := range result {
		assert.Equal(t, 'X', ch)
	}
}

func TestBuffer_ConcurrentReads(t *testing.T) {
	buf := &logr.Buffer{}

	// Write some data
	data := []byte("test data for concurrent reads")
	_, err := buf.Write(data)
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Launch multiple goroutines reading concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			readBuf := make([]byte, 5)
			_, _ = buf.Read(readBuf)
		}()
	}

	wg.Wait()
	// Should not panic or have race conditions
}

func TestBuffer_ConcurrentReadWrite(t *testing.T) {
	buf := &logr.Buffer{}
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = buf.Write([]byte("W"))
			}
		}(i)
	}

	// Readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				readBuf := make([]byte, 1)
				_, _ = buf.Read(readBuf)
			}
		}()
	}

	// String readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = buf.String()
			}
		}()
	}

	wg.Wait()
	// Should not panic or have race conditions
}

func TestBuffer_EmptyRead(t *testing.T) {
	buf := &logr.Buffer{}

	// Try to read from empty buffer
	readBuf := make([]byte, 10)
	n, err := buf.Read(readBuf)
	// EOF is expected when buffer is empty
	assert.Equal(t, 0, n)
	// bytes.Buffer returns EOF, but we just care it doesn't panic
	_ = err
}

func TestBuffer_EmptyString(t *testing.T) {
	buf := &logr.Buffer{}

	result := buf.String()
	assert.Equal(t, "", result)
}
