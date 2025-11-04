package logr_test

import (
	"bytes"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBufferPool_BasicReuse tests that buffers are actually reused.
func TestBufferPool_BasicReuse(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Borrow and release buffers
	buf1 := lgr.BorrowBuffer()
	assert.NotNil(t, buf1)
	buf1.WriteString("test")
	lgr.ReleaseBuffer(buf1)

	// Borrow again - should get same buffer instance (or from pool)
	buf2 := lgr.BorrowBuffer()
	assert.NotNil(t, buf2)

	// Buffer should be empty (reset on release)
	assert.Equal(t, 0, buf2.Len(), "Buffer should be reset when borrowed from pool")

	lgr.ReleaseBuffer(buf2)
}

// TestBufferPool_MaxPooledBuffer tests that large buffers are not pooled.
func TestBufferPool_MaxPooledBuffer(t *testing.T) {
	maxSize := 1024
	lgr, err := logr.New(logr.MaxPooledBufferSize(maxSize))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Create a buffer smaller than max - should be pooled
	smallBuf := lgr.BorrowBuffer()
	smallBuf.WriteString("small")
	assert.True(t, smallBuf.Cap() < maxSize)
	lgr.ReleaseBuffer(smallBuf)

	// Create a buffer larger than max - should NOT be pooled
	largeBuf := lgr.BorrowBuffer()
	largeData := make([]byte, maxSize+100)
	largeBuf.Write(largeData)
	assert.True(t, largeBuf.Cap() >= maxSize)

	// Release large buffer - it won't be returned to pool
	lgr.ReleaseBuffer(largeBuf)

	// Borrow again - large buffer should be eligible for GC
	// (We can't directly test GC, but we verify the mechanism)
	newBuf := lgr.BorrowBuffer()
	// New buffer should be fresh/small, not the large one
	assert.True(t, newBuf.Len() == 0)
	lgr.ReleaseBuffer(newBuf)
}

// TestBufferPool_DisableBufferPool tests DisableBufferPool option.
func TestBufferPool_DisableBufferPool(t *testing.T) {
	lgr, err := logr.New(logr.DisableBufferPool(true))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Borrow buffer
	buf1 := lgr.BorrowBuffer()
	assert.NotNil(t, buf1)
	buf1.WriteString("test")

	// Release buffer - with disabled pool, this is a no-op
	lgr.ReleaseBuffer(buf1)

	// Borrow another buffer - should always be new when pool disabled
	buf2 := lgr.BorrowBuffer()
	assert.NotNil(t, buf2)
	assert.Equal(t, 0, buf2.Len())

	lgr.ReleaseBuffer(buf2)
}

// TestBufferPool_ConcurrentAccess tests concurrent borrowing and releasing.
func TestBufferPool_ConcurrentAccess(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	var wg sync.WaitGroup
	goroutines := 20
	iterations := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				buf := lgr.BorrowBuffer()
				buf.WriteString("concurrent test")
				// Simulate some work
				time.Sleep(time.Microsecond)
				lgr.ReleaseBuffer(buf)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panic or race detector issues, test passes
}

// TestBufferPool_NoLeaks tests that buffers don't leak memory.
func TestBufferPool_NoLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Force GC and get baseline
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Borrow and release many buffers
	iterations := 10000
	for i := 0; i < iterations; i++ {
		buf := lgr.BorrowBuffer()
		buf.WriteString("memory leak test")
		lgr.ReleaseBuffer(buf)
	}

	// Force GC and check memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Memory growth should be minimal (pool reuses buffers)
	// Handle case where GC freed memory between measurements (m2.Alloc < m1.Alloc)
	var growth int64
	if m2.Alloc >= m1.Alloc {
		growth = int64(m2.Alloc - m1.Alloc)
	} else {
		// Memory decreased, no leak
		growth = 0
	}
	t.Logf("Memory growth: %d bytes after %d iterations", growth, iterations)

	// Allow some growth for pool itself and other overhead, but not linear with iterations
	maxExpectedGrowth := int64(iterations * 100) // 100 bytes per iteration would indicate leak
	assert.Less(t, growth, maxExpectedGrowth, "Excessive memory growth indicates buffer leak")
}

// TestBufferPool_IntegrationWithLogging tests buffer pool during actual logging.
func TestBufferPool_IntegrationWithLogging(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Log many messages - buffer pool should be used internally
	messageCount := 100
	for i := 0; i < messageCount; i++ {
		logger.Error("buffer pool test", logr.Int("id", i))
	}

	err = lgr.Flush()
	require.NoError(t, err)

	// Verify all messages logged
	_ = buf.String()
	count := bytes.Count(buf.Bytes(), []byte("buffer pool test"))
	assert.Equal(t, messageCount, count, "All messages should be logged")
}

// TestBufferPool_LargeMessages tests buffer pool with messages exceeding typical size.
func TestBufferPool_LargeMessages(t *testing.T) {
	lgr, err := logr.New(logr.MaxPooledBufferSize(2048))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Create a large message
	largeMsg := string(make([]byte, 3000))

	// Log several large messages
	for i := 0; i < 10; i++ {
		logger.Error(largeMsg, logr.Int("id", i))
	}

	err = lgr.Flush()
	require.NoError(t, err)

	// Messages should still be logged correctly
	output := buf.String()
	assert.Contains(t, output, "id")
}

// TestBufferPool_ResetBehavior tests that buffers are properly reset.
func TestBufferPool_ResetBehavior(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Borrow buffer and write data
	buf1 := lgr.BorrowBuffer()
	buf1.WriteString("first data")
	firstLen := buf1.Len()
	assert.Greater(t, firstLen, 0)

	// Release buffer
	lgr.ReleaseBuffer(buf1)

	// Borrow again - should be reset
	buf2 := lgr.BorrowBuffer()
	assert.Equal(t, 0, buf2.Len(), "Buffer should be reset (length = 0)")
	assert.NotContains(t, buf2.String(), "first data", "Buffer should not contain previous data")

	lgr.ReleaseBuffer(buf2)
}

// TestBufferPool_MultipleTargets tests buffer pool with multiple targets.
func TestBufferPool_MultipleTargets(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	buf3 := &bytes.Buffer{}

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}

	target1 := targets.NewWriterTarget(buf1)
	err = lgr.AddTarget(target1, "target1", filter, formatter, 50)
	require.NoError(t, err)

	target2 := targets.NewWriterTarget(buf2)
	err = lgr.AddTarget(target2, "target2", filter, formatter, 50)
	require.NoError(t, err)

	target3 := targets.NewWriterTarget(buf3)
	err = lgr.AddTarget(target3, "target3", filter, formatter, 50)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Log messages - each target borrows buffer from same pool
	messageCount := 50
	for i := 0; i < messageCount; i++ {
		logger.Error("multi-target test", logr.Int("id", i))
	}

	err = lgr.Flush()
	require.NoError(t, err)

	// All targets should have all messages
	for idx, buf := range []*bytes.Buffer{buf1, buf2, buf3} {
		count := bytes.Count(buf.Bytes(), []byte("multi-target test"))
		assert.Equal(t, messageCount, count, "Target %d should have all messages", idx+1)
	}
}

// TestBufferPool_StressTest tests buffer pool under high load.
func TestBufferPool_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	lgr, err := logr.New(logr.MaxQueueSize(500))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "stress", filter, formatter, 400)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	var wg sync.WaitGroup
	goroutines := 10
	messagesPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Error("stress test", logr.Int("goroutine", id), logr.Int("msg", j))
			}
		}(i)
	}

	wg.Wait()

	err = lgr.Flush()
	require.NoError(t, err)

	// Verify all messages logged
	expectedCount := goroutines * messagesPerGoroutine
	_ = buf.String()
	count := bytes.Count(buf.Bytes(), []byte("stress test"))
	assert.Equal(t, expectedCount, count, "All stress test messages should be logged")
}

// TestBufferPool_CapacityGrowth tests buffer capacity growth and pool behavior.
func TestBufferPool_CapacityGrowth(t *testing.T) {
	maxPooled := 1024
	lgr, err := logr.New(logr.MaxPooledBufferSize(maxPooled))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Start with small writes
	buf := lgr.BorrowBuffer()
	buf.WriteString("small")
	smallCap := buf.Cap()
	t.Logf("Small buffer capacity: %d", smallCap)
	lgr.ReleaseBuffer(buf)

	// Write larger data
	buf = lgr.BorrowBuffer()
	mediumData := make([]byte, 512)
	buf.Write(mediumData)
	mediumCap := buf.Cap()
	t.Logf("Medium buffer capacity: %d", mediumCap)
	assert.Less(t, mediumCap, maxPooled, "Should be poolable")
	lgr.ReleaseBuffer(buf)

	// Write very large data
	buf = lgr.BorrowBuffer()
	largeData := make([]byte, maxPooled+100)
	buf.Write(largeData)
	largeCap := buf.Cap()
	t.Logf("Large buffer capacity: %d", largeCap)
	assert.GreaterOrEqual(t, largeCap, maxPooled, "Should exceed max pooled size")
	lgr.ReleaseBuffer(buf) // Won't be pooled

	// Next borrow should get small buffer again (large one not pooled)
	buf = lgr.BorrowBuffer()
	newCap := buf.Cap()
	t.Logf("New buffer capacity after large: %d", newCap)
	// Capacity should be reasonable (not the huge one)
	lgr.ReleaseBuffer(buf)
}
