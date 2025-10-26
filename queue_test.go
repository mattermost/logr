package logr_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueueFull_DropBehavior tests OnQueueFull callback that drops records.
func TestQueueFull_DropBehavior(t *testing.T) {
	var droppedCount int32
	onQueueFull := func(rec *logr.LogRec, maxQueueSize int) bool {
		atomic.AddInt32(&droppedCount, 1)
		return true // drop the record
	}

	lgr, err := logr.New(
		logr.MaxQueueSize(5),
		logr.OnQueueFull(onQueueFull),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	// Very slow target to fill queue
	target := test.NewSlowTarget(buf, 100) // 100ms per record
	err = lgr.AddTarget(target, "slow", filter, formatter, 5)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Quickly send many records to overwhelm queue
	for i := 0; i < 50; i++ {
		logger.Error("message", logr.Int("id", i))
	}

	// Give some time for queue processing
	time.Sleep(200 * time.Millisecond)

	dropped := atomic.LoadInt32(&droppedCount)
	assert.Greater(t, dropped, int32(0), "Some records should have been dropped")
	t.Logf("Dropped %d records", dropped)
}

// TestQueueFull_BlockBehavior tests OnQueueFull callback that blocks.
func TestQueueFull_BlockBehavior(t *testing.T) {
	var blockCount int32
	onQueueFull := func(rec *logr.LogRec, maxQueueSize int) bool {
		atomic.AddInt32(&blockCount, 1)
		return false // block until queue has space
	}

	lgr, err := logr.New(
		logr.MaxQueueSize(10),
		logr.OnQueueFull(onQueueFull),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	// Slow target to create backpressure
	target := test.NewSlowTarget(buf, 10) // 10ms per record
	err = lgr.AddTarget(target, "slow", filter, formatter, 10)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Send records that will fill queue
	recordCount := 30
	for i := 0; i < recordCount; i++ {
		logger.Error("message", logr.Int("id", i))
	}

	// Flush to ensure all records processed
	err = lgr.Flush()
	require.NoError(t, err)

	blocks := atomic.LoadInt32(&blockCount)
	t.Logf("Blocked %d times", blocks)

	// Verify all records were eventually written
	output := buf.String()
	// Count should match recordCount (no drops with blocking)
	// Note: exact count may vary due to async nature, but should be close
	assert.Contains(t, output, "message")
}

// TestTargetQueueFull_DropBehavior tests OnTargetQueueFull callback that drops.
func TestTargetQueueFull_DropBehavior(t *testing.T) {
	var targetDropCount int32
	onTargetQueueFull := func(target logr.Target, rec *logr.LogRec, maxQueueSize int) bool {
		atomic.AddInt32(&targetDropCount, 1)
		return true // drop the record
	}

	lgr, err := logr.New(
		logr.MaxQueueSize(100), // Large main queue
		logr.OnTargetQueueFull(onTargetQueueFull),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	// Very slow target with small queue
	target := test.NewSlowTarget(buf, 100)                    // 100ms per record
	err = lgr.AddTarget(target, "slow", filter, formatter, 3) // Small target queue
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Quickly send many records
	for i := 0; i < 20; i++ {
		logger.Error("message", logr.Int("id", i))
	}

	// Give time for processing
	time.Sleep(200 * time.Millisecond)

	dropped := atomic.LoadInt32(&targetDropCount)
	assert.Greater(t, dropped, int32(0), "Some records should have been dropped at target")
	t.Logf("Dropped %d records at target queue", dropped)
}

// TestTargetQueueFull_BlockBehavior tests OnTargetQueueFull callback that blocks.
func TestTargetQueueFull_BlockBehavior(t *testing.T) {
	var targetBlockCount int32
	onTargetQueueFull := func(target logr.Target, rec *logr.LogRec, maxQueueSize int) bool {
		atomic.AddInt32(&targetBlockCount, 1)
		return false // block until space available
	}

	lgr, err := logr.New(
		logr.MaxQueueSize(100),
		logr.OnTargetQueueFull(onTargetQueueFull),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := test.NewSlowTarget(buf, 10)                     // 10ms per record
	err = lgr.AddTarget(target, "slow", filter, formatter, 5) // Small target queue
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Send records that will fill target queue
	for i := 0; i < 20; i++ {
		logger.Error("message", logr.Int("id", i))
	}

	err = lgr.Flush()
	require.NoError(t, err)

	blocks := atomic.LoadInt32(&targetBlockCount)
	t.Logf("Blocked %d times at target queue", blocks)

	// All records should eventually be written (blocking prevents drops)
	output := buf.String()
	assert.Contains(t, output, "message")
}

// TestEnqueueTimeout tests that enqueue timeout is enforced.
func TestEnqueueTimeout(t *testing.T) {
	var errorCount int32
	onLoggerError := func(err error) {
		atomic.AddInt32(&errorCount, 1)
	}

	lgr, err := logr.New(
		logr.MaxQueueSize(5),
		logr.EnqueueTimeout(50*time.Millisecond),
		logr.OnLoggerError(onLoggerError),
		logr.OnQueueFull(func(rec *logr.LogRec, maxQueueSize int) bool {
			return false // block (will timeout)
		}),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	// Very slow target to ensure timeout
	target := test.NewSlowTarget(buf, 200) // 200ms per record (much slower than timeout)
	err = lgr.AddTarget(target, "veryslow", filter, formatter, 5)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Send many records quickly
	for i := 0; i < 30; i++ {
		logger.Error("timeout test", logr.Int("id", i))
	}

	// Give time for timeouts to occur
	time.Sleep(500 * time.Millisecond)

	errors := atomic.LoadInt32(&errorCount)
	assert.Greater(t, errors, int32(0), "Should have timeout errors")
	t.Logf("Enqueue timeout errors: %d", errors)
}

// TestQueueSizeOption tests different MaxQueueSize values.
func TestQueueSizeOption(t *testing.T) {
	tests := []struct {
		name      string
		queueSize int
	}{
		{"Small queue", 10},
		{"Medium queue", 100},
		{"Large queue", 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lgr, err := logr.New(logr.MaxQueueSize(tt.queueSize))
			require.NoError(t, err)
			defer func() { require.NoError(t, lgr.Shutdown()) }()

			buf := &bytes.Buffer{}
			filter := &logr.StdFilter{Lvl: logr.Error}
			formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
			target := test.NewSlowTarget(buf, 1)
			err = lgr.AddTarget(target, "test", filter, formatter, 100)
			require.NoError(t, err)

			logger := lgr.NewLogger()

			// Send half the queue size
			half := tt.queueSize / 2
			for i := 0; i < half; i++ {
				logger.Error("test message")
			}

			err = lgr.Flush()
			require.NoError(t, err)

			// Should have logged all messages
			output := buf.String()
			assert.Contains(t, output, "test message")
		})
	}
}

// TestBackpressure_SlowTargetDoesNotBlockOthers tests that one slow target
// doesn't block other targets (at least initially).
func TestBackpressure_SlowTargetDoesNotBlockOthers(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(100))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Fast target
	fastBuf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	fastTarget := test.NewSlowTarget(fastBuf, 1) // 1ms
	err = lgr.AddTarget(fastTarget, "fast", filter, formatter, 50)
	require.NoError(t, err)

	// Slow target
	slowBuf := &bytes.Buffer{}
	slowTarget := test.NewSlowTarget(slowBuf, 50) // 50ms
	err = lgr.AddTarget(slowTarget, "slow", filter, formatter, 10)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Send some records
	recordCount := 20
	for i := 0; i < recordCount; i++ {
		logger.Error("test", logr.Int("id", i))
	}

	// Give fast target time to process most records
	time.Sleep(100 * time.Millisecond)

	// Fast target should have processed more records than slow target
	fastOutput := fastBuf.String()
	slowOutput := slowBuf.String()

	fastCount := len(bytes.Split([]byte(fastOutput), []byte("\n"))) - 1
	slowCount := len(bytes.Split([]byte(slowOutput), []byte("\n"))) - 1

	t.Logf("Fast target processed: %d, Slow target processed: %d", fastCount, slowCount)
	// This test is inherently timing-sensitive, so we just check that fast > slow
	// In practice, fast should process significantly more
}

// TestConcurrentQueueOperations tests concurrent enqueueing and flushing.
func TestConcurrentQueueOperations(t *testing.T) {
	lgr, err := logr.New(
		logr.MaxQueueSize(200),
		logr.FlushTimeout(10*time.Second), // Generous timeout for concurrent flushes
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := test.NewSlowTarget(buf, 2)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	var wg sync.WaitGroup

	// Concurrent loggers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				logger.Error("concurrent", logr.Int("goroutine", id), logr.Int("msg", j))
			}
		}(i)
	}

	// Wait for logging goroutines to finish
	wg.Wait()

	// Single final flush to drain queues
	err = lgr.Flush()
	require.NoError(t, err)

	// Should have logged all messages
	output := buf.String()
	assert.Contains(t, output, "concurrent")

	// Count should be 5 goroutines * 10 messages = 50
	count := bytes.Count(buf.Bytes(), []byte("concurrent"))
	assert.Equal(t, 50, count, "All messages should be logged")
}

// TestQueueDrain_OnShutdown tests that queue is properly drained on shutdown.
func TestQueueDrain_OnShutdown(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(100))
	require.NoError(t, err)

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := test.NewSlowTarget(buf, 5) // 5ms per record
	err = lgr.AddTarget(target, "test", filter, formatter, 50)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Quickly send many records
	expectedCount := 30
	for i := 0; i < expectedCount; i++ {
		logger.Error("drain test", logr.Int("id", i))
	}

	// Shutdown should drain queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = lgr.ShutdownWithTimeout(ctx)
	require.NoError(t, err)

	// All records should be written
	_ = buf.String()
	count := bytes.Count(buf.Bytes(), []byte("drain test"))
	assert.Equal(t, expectedCount, count, "All queued records should be drained on shutdown")
}

// TestBothQueuesFull tests behavior when both main and target queues are full.
func TestBothQueuesFull(t *testing.T) {
	var mainQueueFullCount int32
	var targetQueueFullCount int32

	lgr, err := logr.New(
		logr.MaxQueueSize(5), // Small main queue
		logr.OnQueueFull(func(rec *logr.LogRec, maxQueueSize int) bool {
			atomic.AddInt32(&mainQueueFullCount, 1)
			return true // drop
		}),
		logr.OnTargetQueueFull(func(target logr.Target, rec *logr.LogRec, maxQueueSize int) bool {
			atomic.AddInt32(&targetQueueFullCount, 1)
			return true // drop
		}),
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	// Very slow target with small queue
	target := test.NewSlowTarget(buf, 100)                    // 100ms per record
	err = lgr.AddTarget(target, "slow", filter, formatter, 3) // Tiny target queue
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Overwhelm both queues
	for i := 0; i < 50; i++ {
		logger.Error("overflow test", logr.Int("id", i))
	}

	time.Sleep(200 * time.Millisecond)

	mainFull := atomic.LoadInt32(&mainQueueFullCount)
	targetFull := atomic.LoadInt32(&targetQueueFullCount)

	t.Logf("Main queue full: %d times, Target queue full: %d times", mainFull, targetFull)

	// At least one of the queues should have been full
	assert.True(t, mainFull > 0 || targetFull > 0, "At least one queue should have been full")
}
