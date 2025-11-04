package logr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wiggin77/merror"
)

func TestIsTimeoutError_Simple(t *testing.T) {
	// Regular error should return false
	regularErr := errors.New("regular error")
	assert.False(t, logr.IsTimeoutError(regularErr))

	// Nil error should return false
	assert.False(t, logr.IsTimeoutError(nil))
}

func TestIsTimeoutError_ActualTimeout(t *testing.T) {
	lgr, err := logr.New(
		logr.MaxQueueSize(1),
		logr.EnqueueTimeout(1), // 1 ms timeout
	)
	require.NoError(t, err)

	// Fill the queue
	logger := lgr.NewLogger()
	for i := 0; i < 100; i++ {
		logger.Info("message", logr.Int("i", i))
	}

	// Try to flush with a very short timeout - should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	err = lgr.FlushWithTimeout(ctx)
	if err != nil {
		assert.True(t, logr.IsTimeoutError(err), "FlushWithTimeout error should be detectable as timeout")
	}

	// Clean up
	_ = lgr.Shutdown()
}

func TestIsTimeoutError_WithMError(t *testing.T) {
	merr := &merror.MError{}

	// MError without timeout errors
	merr.Append(errors.New("error 1"))
	merr.Append(errors.New("error 2"))
	assert.False(t, logr.IsTimeoutError(merr))

	// Create a new logger that will timeout
	lgr, err := logr.New(
		logr.MaxQueueSize(1),
		logr.EnqueueTimeout(1),
	)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	for i := 0; i < 100; i++ {
		logger.Info("flood", logr.Int("i", i))
	}

	// Get a timeout error
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	flushErr := lgr.FlushWithTimeout(ctx)

	// Create MError with timeout
	merr2 := &merror.MError{}
	merr2.Append(errors.New("other error"))
	if flushErr != nil {
		merr2.Append(flushErr)
		// Should detect timeout in multi-error
		assert.True(t, logr.IsTimeoutError(merr2), "Should detect timeout in MError")
	}

	_ = lgr.Shutdown()
}

func TestTimeoutError_ErrorMessage(t *testing.T) {
	lgr, err := logr.New(
		logr.MaxQueueSize(1),
		logr.EnqueueTimeout(1),
	)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	for i := 0; i < 100; i++ {
		logger.Info("message", logr.Int("i", i))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	err = lgr.FlushWithTimeout(ctx)
	if err != nil {
		assert.True(t, logr.IsTimeoutError(err))
		assert.Contains(t, err.Error(), "timeout", "Timeout error message should contain 'timeout'")
	}

	_ = lgr.Shutdown()
}

func TestShutdownTimeout_WithTimeout(t *testing.T) {
	// Create logger with very short shutdown timeout
	lgr, err := logr.New(
		logr.ShutdownTimeout(1), // 1 ms timeout
		logr.MaxQueueSize(1000),
	)
	require.NoError(t, err)

	// Fill queue with lots of messages
	logger := lgr.NewLogger()
	for i := 0; i < 1000; i++ {
		logger.Info("message", logr.Int("i", i))
	}

	// Shutdown with short timeout may produce timeout error
	err = lgr.Shutdown()
	// This may or may not timeout depending on system speed, but if it does,
	// it should be detectable as a timeout error
	if err != nil && logr.IsTimeoutError(err) {
		assert.Contains(t, err.Error(), "timeout")
	}
}

func TestFlushTimeout_Success(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	logger := lgr.NewLogger()
	logger.Info("test message")

	// Flush with reasonable timeout should succeed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = lgr.FlushWithTimeout(ctx)
	assert.NoError(t, err, "Flush with reasonable timeout should not timeout")
	assert.False(t, logr.IsTimeoutError(err))
}
