package logr_test

import (
	"bytes"
	"context"
	"fmt"
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

// TestTargetShutdownQueueDrain tests that all queued records are processed during shutdown
func TestTargetShutdownQueueDrain(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	
	// Use slow target to ensure records are queued
	target := test.NewSlowTarget(buf, 10) // 10ms delay per record
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target, "drainTest", filter, formatter, 100)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	// Send multiple records quickly to queue them up
	expectedRecords := 10
	for i := 0; i < expectedRecords; i++ {
		logger.Info("Test record", logr.Int("id", i))
	}
	
	// Shutdown with generous timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = lgr.ShutdownWithTimeout(ctx)
	assert.NoError(t, err)
	
	// Verify all records were written
	output := buf.String()
	for i := 0; i < expectedRecords; i++ {
		assert.Contains(t, output, "Test record", fmt.Sprintf("Missing record %d", i))
	}
	
	// Count actual records written
	lines := bytes.Count(buf.Bytes(), []byte("Test record"))
	assert.Equal(t, expectedRecords, lines, "Not all records were drained")
}

// TestTargetShutdownRaceCondition tests prevention of race between Log() and Shutdown()
func TestTargetShutdownRaceCondition(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(buf, 1) // 1ms delay
	
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target, "raceTest", filter, formatter, 50)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	var logAttempts int32
	var wg sync.WaitGroup
	
	// Start goroutines that continuously try to log
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				atomic.AddInt32(&logAttempts, 1)
				logger.Info("Race test", logr.Int("goroutine", id), logr.Int("attempt", j))
				time.Sleep(time.Millisecond) // Small delay to increase race window
			}
		}(i)
	}
	
	// Wait a bit for logging to start, then shutdown
	time.Sleep(10 * time.Millisecond)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err = lgr.ShutdownWithTimeout(ctx)
	assert.NoError(t, err)
	
	wg.Wait() // Wait for all goroutines to complete
	
	// Verify no panics occurred and some logs were written
	output := buf.String()
	assert.Contains(t, output, "Race test")
	
	totalAttempts := atomic.LoadInt32(&logAttempts)
	t.Logf("Total log attempts: %d", totalAttempts)
	assert.Greater(t, totalAttempts, int32(0))
}

// TestTargetShutdownContextTimeout tests behavior when context timeout is reached
func TestTargetShutdownContextTimeout(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	
	// Use very slow target to force timeout
	target := test.NewSlowTarget(buf, 100) // 100ms delay per record
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target, "timeoutTest", filter, formatter, 100)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	// Queue many records that will take a long time to process
	for i := 0; i < 20; i++ {
		logger.Info("Timeout test record", logr.Int("id", i))
	}
	
	// Use short timeout to force early exit
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	err = lgr.ShutdownWithTimeout(ctx)
	duration := time.Since(start)
	
	// Shutdown should complete quickly due to timeout  
	// Flush timeout is expected behavior during short timeout scenarios
	assert.Error(t, err, "Expected flush timeout error during short timeout")
	assert.Less(t, int64(duration), int64(200*time.Millisecond), "Shutdown took too long")
	
	// Some records might be processed, but not all due to timeout
	recordCount := bytes.Count(buf.Bytes(), []byte("Timeout test record"))
	t.Logf("Records processed before timeout: %d/20", recordCount)
	assert.LessOrEqual(t, recordCount, 20)
}

// TestTargetShutdownMultipleTargets tests shutdown with multiple targets
func TestTargetShutdownMultipleTargets(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	
	target1 := test.NewSlowTarget(buf1, 5)  // 5ms delay
	target2 := test.NewSlowTarget(buf2, 10) // 10ms delay
	
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target1, "target1", filter, formatter, 50)
	require.NoError(t, err)
	
	err = lgr.AddTarget(target2, "target2", filter, formatter, 50)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	// Send records to both targets
	expectedRecords := 15
	for i := 0; i < expectedRecords; i++ {
		logger.Info("Multi-target test", logr.Int("id", i))
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = lgr.ShutdownWithTimeout(ctx)
	assert.NoError(t, err)
	
	// Verify both targets received all records
	count1 := bytes.Count(buf1.Bytes(), []byte("Multi-target test"))
	count2 := bytes.Count(buf2.Bytes(), []byte("Multi-target test"))
	
	assert.Equal(t, expectedRecords, count1, "Target1 missing records")
	assert.Equal(t, expectedRecords, count2, "Target2 missing records")
	assert.Contains(t, buf1.String(), "Multi-target test")
	assert.Contains(t, buf2.String(), "Multi-target test")
}

// TestTargetShutdownFlushRecords tests that flush records are handled during drain
func TestTargetShutdownFlushRecords(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(buf, 5) // 5ms delay
	
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target, "flushTest", filter, formatter, 50)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	// Send some records
	for i := 0; i < 5; i++ {
		logger.Info("Before flush", logr.Int("id", i))
	}
	
	// Trigger a flush which will add flush records to queue
	go func() {
		time.Sleep(10 * time.Millisecond)
		err := lgr.Flush()
		assert.NoError(t, err)
	}()
	
	// Add more records after flush
	for i := 5; i < 10; i++ {
		logger.Info("After flush", logr.Int("id", i))
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	err = lgr.ShutdownWithTimeout(ctx)
	assert.NoError(t, err)
	
	// Verify all regular records were processed (flush records are internal)
	beforeCount := bytes.Count(buf.Bytes(), []byte("Before flush"))
	afterCount := bytes.Count(buf.Bytes(), []byte("After flush"))
	
	assert.Equal(t, 5, beforeCount, "Missing 'before flush' records")
	assert.Equal(t, 5, afterCount, "Missing 'after flush' records")
}

// TestTargetShutdownEmptyQueue tests shutdown behavior with empty queue
func TestTargetShutdownEmptyQueue(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(buf, 1)
	
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target, "emptyTest", filter, formatter, 10)
	require.NoError(t, err)
	
	// Don't send any records, just shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	start := time.Now()
	err = lgr.ShutdownWithTimeout(ctx)
	duration := time.Since(start)
	
	assert.NoError(t, err)
	assert.Less(t, int64(duration), int64(100*time.Millisecond), "Empty queue shutdown took too long")
	assert.Empty(t, buf.String(), "Buffer should be empty")
}

// TestDrainQueueRespectsTimeout specifically tests that drainQueue respects the shutdown context timeout
func TestDrainQueueRespectsTimeout(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	
	// Use extremely slow target to ensure drainQueue will timeout
	target := test.NewSlowTarget(buf, 200) // 200ms delay per record
	lgr, err := logr.New()
	require.NoError(t, err)
	
	// Small queue to ensure records get queued
	err = lgr.AddTarget(target, "drainTimeoutTest", filter, formatter, 5)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	// Queue multiple records quickly
	for i := 0; i < 3; i++ {
		logger.Info("Drain timeout test", logr.Int("id", i))
	}
	
	// Give a moment for records to be queued
	time.Sleep(10 * time.Millisecond)
	
	// Use very short timeout - shorter than it would take to process even 1 record
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	start := time.Now()
	err = lgr.ShutdownWithTimeout(ctx)
	duration := time.Since(start)
	
	// Shutdown should complete quickly due to drainQueue timeout
	// Flush timeout is expected behavior during short timeout scenarios
	assert.Error(t, err, "Expected flush timeout error during drainQueue timeout")
	assert.Less(t, int64(duration), int64(100*time.Millisecond), "drainQueue did not respect timeout")
	
	// Should have processed very few or no records due to timeout
	recordCount := bytes.Count(buf.Bytes(), []byte("Drain timeout test"))
	t.Logf("Records processed before drainQueue timeout: %d/3", recordCount)
	assert.LessOrEqual(t, recordCount, 1, "Too many records processed, drainQueue may not be respecting timeout")
}

// TestDrainQueueWithoutTimeout tests that drainQueue processes all records when there's sufficient time
func TestDrainQueueWithoutTimeout(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	
	// Use moderately slow target
	target := test.NewSlowTarget(buf, 20) // 20ms delay per record
	lgr, err := logr.New()
	require.NoError(t, err)
	
	err = lgr.AddTarget(target, "drainSuccessTest", filter, formatter, 10)
	require.NoError(t, err)
	
	logger := lgr.NewLogger()
	
	// Queue records
	expectedRecords := 5
	for i := 0; i < expectedRecords; i++ {
		logger.Info("Drain success test", logr.Int("id", i))
	}
	
	// Give time for records to be queued
	time.Sleep(10 * time.Millisecond)
	
	// Use generous timeout - more than enough to process all records
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	err = lgr.ShutdownWithTimeout(ctx)
	assert.NoError(t, err)
	
	// All records should be processed when timeout is sufficient
	recordCount := bytes.Count(buf.Bytes(), []byte("Drain success test"))
	assert.Equal(t, expectedRecords, recordCount, "drainQueue should process all records when timeout is sufficient")
}