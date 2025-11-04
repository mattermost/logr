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
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrent_IsLevelEnabled tests concurrent IsLevelEnabled calls.
func TestConcurrent_IsLevelEnabled(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	var wg sync.WaitGroup
	goroutines := 20
	checksPerGoroutine := 1000

	// Multiple goroutines checking different levels concurrently
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < checksPerGoroutine; j++ {
				// Rotate through different levels
				levels := []logr.Level{logr.Error, logr.Warn, logr.Info, logr.Debug, logr.Trace}
				lvl := levels[j%len(levels)]
				_ = lgr.IsLevelEnabled(lvl)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panic or race, test passes
}

// TestConcurrent_NewLogger tests concurrent Logger creation.
func TestConcurrent_NewLogger(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	var wg sync.WaitGroup
	goroutines := 50
	loggersPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < loggersPerGoroutine; j++ {
				logger := lgr.NewLogger()
				_ = logger.With(logr.Int("id", id), logr.Int("iter", j))
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrent_NewLoggerWithLogging tests creating loggers and logging concurrently.
func TestConcurrent_NewLoggerWithLogging(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(500))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 400)
	require.NoError(t, err)

	var wg sync.WaitGroup
	goroutines := 20
	messagesPerGoroutine := 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger := lgr.NewLogger().With(logr.Int("goroutine", id))
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Error("concurrent logging", logr.Int("msg", j))
			}
		}(i)
	}

	wg.Wait()

	err = lgr.Flush()
	require.NoError(t, err)

	// Verify all messages logged
	expectedCount := goroutines * messagesPerGoroutine
	_ = buf.String()
	count := bytes.Count(buf.Bytes(), []byte("concurrent logging"))
	assert.Equal(t, expectedCount, count, "All concurrent messages should be logged")
}

// TestConcurrent_AddTarget tests adding targets while logging.
func TestConcurrent_AddTarget(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(200))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Initial target
	buf1 := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target1 := targets.NewWriterTarget(buf1)
	err = lgr.AddTarget(target1, "initial", filter, formatter, 150)
	require.NoError(t, err)

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutine continuously logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := lgr.NewLogger()
		for {
			select {
			case <-stopCh:
				return
			default:
				logger.Error("continuous logging")
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// Add multiple targets concurrently while logging
	targetCount := 5
	for i := 0; i < targetCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond) // Stagger additions
			buf := &bytes.Buffer{}
			target := targets.NewWriterTarget(buf)
			err := lgr.AddTarget(target, "dynamic"+string(rune(id)), filter, formatter, 50)
			assert.NoError(t, err)
		}(i)
	}

	// Let everything run
	time.Sleep(100 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	err = lgr.Flush()
	require.NoError(t, err)

	// Should have logged messages
	output := buf1.String()
	assert.Contains(t, output, "continuous logging")
}

// TestConcurrent_RemoveTarget tests removing targets while logging.
func TestConcurrent_RemoveTarget(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(200))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Add multiple targets
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}

	for i := 0; i < 5; i++ {
		buf := &bytes.Buffer{}
		target := targets.NewWriterTarget(buf)
		err = lgr.AddTarget(target, "target"+string(rune('A'+i)), filter, formatter, 50)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutine continuously logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := lgr.NewLogger()
		for {
			select {
			case <-stopCh:
				return
			default:
				logger.Error("logging during removal")
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// Remove targets concurrently
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
			targetName := "target" + string(rune('A'+id))
			err := lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
				return ti.Name == targetName
			})
			assert.NoError(t, err)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	// Should still have 2 targets
	infos := lgr.TargetInfos()
	assert.Equal(t, 2, len(infos), "Should have 2 targets remaining")
}

// TestConcurrent_AddAndRemoveTargets tests adding and removing targets simultaneously.
func TestConcurrent_AddAndRemoveTargets(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(200))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}

	// Initial targets
	for i := 0; i < 3; i++ {
		buf := &bytes.Buffer{}
		target := targets.NewWriterTarget(buf)
		err = lgr.AddTarget(target, "initial"+string(rune('A'+i)), filter, formatter, 50)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	var addCounter, removeCounter int32

	// Goroutine continuously logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger := lgr.NewLogger()
		for {
			select {
			case <-stopCh:
				return
			default:
				logger.Error("during churn")
				time.Sleep(time.Millisecond)
			}
		}
	}()

	// Goroutines adding targets
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				select {
				case <-stopCh:
					return
				default:
					buf := &bytes.Buffer{}
					target := targets.NewWriterTarget(buf)
					targetName := "dynamic" + string(rune('A'+id*10+j))
					err := lgr.AddTarget(target, targetName, filter, formatter, 50)
					if err == nil {
						atomic.AddInt32(&addCounter, 1)
					}
					time.Sleep(5 * time.Millisecond)
				}
			}
		}(i)
	}

	// Goroutines removing targets
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(20 * time.Millisecond) // Let some targets be added first
			for j := 0; j < 2; j++ {
				select {
				case <-stopCh:
					return
				default:
					targetName := "dynamic" + string(rune('A'+id*10+j))
					err := lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
						return ti.Name == targetName
					})
					if err == nil {
						atomic.AddInt32(&removeCounter, 1)
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	time.Sleep(200 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	added := atomic.LoadInt32(&addCounter)
	removed := atomic.LoadInt32(&removeCounter)
	t.Logf("Added: %d targets, Removed: %d targets", added, removed)

	// Should have some targets remaining
	infos := lgr.TargetInfos()
	assert.Greater(t, len(infos), 0, "Should have at least some targets")
}

// TestConcurrent_IsLevelEnabledWithCacheReset tests IsLevelEnabled during cache resets.
func TestConcurrent_IsLevelEnabledWithCacheReset(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutines checking levels
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = lgr.IsLevelEnabled(logr.Error)
					_ = lgr.IsLevelEnabled(logr.Info)
					_ = lgr.IsLevelEnabled(logr.Debug)
				}
			}
		}()
	}

	// Goroutine resetting cache
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case <-stopCh:
				return
			default:
				lgr.ResetLevelCache()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	time.Sleep(200 * time.Millisecond)
	close(stopCh)
	wg.Wait()
	// If no panic or race, test passes
}

// TestConcurrent_LoggerWithFields tests concurrent Logger.With() calls.
func TestConcurrent_LoggerWithFields(t *testing.T) {
	lgr, err := logr.New(logr.MaxQueueSize(500))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 400)
	require.NoError(t, err)

	baseLogger := lgr.NewLogger()

	var wg sync.WaitGroup
	goroutines := 20

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Create logger chain
			logger := baseLogger.With(logr.Int("goroutine", id))
			logger = logger.With(logr.String("level", "first"))
			logger = logger.With(logr.String("level", "second"))

			// Log with additional fields
			for j := 0; j < 10; j++ {
				logger.Error("chained", logr.Int("msg", j))
			}
		}(i)
	}

	wg.Wait()

	err = lgr.Flush()
	require.NoError(t, err)

	// Verify all messages logged
	expectedCount := goroutines * 10
	_ = buf.String()
	count := bytes.Count(buf.Bytes(), []byte("chained"))
	assert.Equal(t, expectedCount, count, "All messages should be logged")
}

// TestConcurrent_FlushWhileLogging tests concurrent Flush calls while logging.
func TestConcurrent_FlushWhileLogging(t *testing.T) {
	lgr, err := logr.New(
		logr.MaxQueueSize(300),
		logr.FlushTimeout(5*time.Second), // Increase timeout for concurrent flushes
	)
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 250)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutines logging
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					logger.Error("flush test", logr.Int("goroutine", id))
					time.Sleep(time.Millisecond)
				}
			}
		}(i)
	}

	// Goroutines flushing - space them out more to avoid overwhelming the flush mechanism
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Stagger the start of each flusher
			time.Sleep(time.Duration(id*30) * time.Millisecond)
			for j := 0; j < 3; j++ {
				select {
				case <-stopCh:
					return
				default:
					err := lgr.Flush()
					assert.NoError(t, err)
					time.Sleep(50 * time.Millisecond) // More time between flushes
				}
			}
		}(i)
	}

	time.Sleep(1 * time.Second)
	close(stopCh)
	wg.Wait()

	err = lgr.Flush()
	require.NoError(t, err)

	// Should have logged messages
	output := buf.String()
	assert.Contains(t, output, "flush test")
}

// TestConcurrent_LoggerCreationStress tests stress on logger creation.
func TestConcurrent_LoggerCreationStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	var wg sync.WaitGroup
	goroutines := 100
	loggersPerGoroutine := 1000

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < loggersPerGoroutine; j++ {
				logger := lgr.NewLogger()
				logger = logger.With(logr.Int("id", id))
				_ = logger.With(logr.Int("iter", j))
				// Logger created but not necessarily used
			}
		}(i)
	}

	wg.Wait()
	t.Logf("Created %d loggers across %d goroutines", goroutines*loggersPerGoroutine, goroutines)
}
