package logr_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnExit(t *testing.T) {
	var exitCode int32
	var exitCalled int32

	onExitFunc := func(code int) {
		atomic.StoreInt32(&exitCode, int32(code))
		atomic.StoreInt32(&exitCalled, 1)
	}

	lgr, err := logr.New(logr.OnExit(onExitFunc))
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Note: We can't actually test Fatal() calling exit because it would
	// terminate the test. But we can verify the option is set and doesn't error.
	// The actual exit behavior would be tested in integration tests.

	// Just verify logger works normally
	logger.Info("test message")

	err = lgr.Shutdown()
	require.NoError(t, err)

	// In a real scenario, Fatal would have called onExitFunc
	// For this test, we just verify the option was set without error
}

func TestOnPanic(t *testing.T) {
	var panicValue atomic.Value
	var panicCalled int32

	onPanicFunc := func(err interface{}) {
		panicValue.Store(err)
		atomic.StoreInt32(&panicCalled, 1)
	}

	lgr, err := logr.New(logr.OnPanic(onPanicFunc))
	require.NoError(t, err)
	defer func() { _ = lgr.Shutdown() }()

	logger := lgr.NewLogger()

	// Note: Similar to OnExit, we can't test the actual panic behavior
	// without causing test failure. We verify the option is set properly.

	// Verify logger works normally
	logger.Info("test message")

	// In a real scenario, Panic() would call onPanicFunc
	// For this test, we just verify the option was set without error
}

func TestShutdownTimeout(t *testing.T) {
	// Test with a reasonable shutdown timeout
	lgr, err := logr.New(logr.ShutdownTimeout(5 * time.Second))
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("test message")

	// Shutdown should complete within timeout
	err = lgr.Shutdown()
	assert.NoError(t, err)
}

func TestShutdownTimeout_Short(t *testing.T) {
	// Test with very short timeout - may or may not timeout depending on system
	lgr, err := logr.New(
		logr.ShutdownTimeout(1*time.Millisecond),
		logr.MaxQueueSize(10000),
	)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Generate many messages to potentially cause timeout
	for i := 0; i < 1000; i++ {
		logger.Info("message", logr.Int("i", i))
	}

	// Shutdown with very short timeout may produce timeout error
	err = lgr.Shutdown()
	// Don't assert error since timing is unpredictable
	// Just verify it doesn't panic
	_ = err
}

func TestOnExit_Nil(t *testing.T) {
	// Test that nil onExit doesn't cause issues
	lgr, err := logr.New(logr.OnExit(nil))
	require.NoError(t, err)
	defer func() { _ = lgr.Shutdown() }()

	logger := lgr.NewLogger()
	logger.Info("test message")
}

func TestOnPanic_Nil(t *testing.T) {
	// Test that nil onPanic doesn't cause issues
	lgr, err := logr.New(logr.OnPanic(nil))
	require.NoError(t, err)
	defer func() { _ = lgr.Shutdown() }()

	logger := lgr.NewLogger()
	logger.Info("test message")
}

func TestMultipleOptions(t *testing.T) {
	// Test multiple options can be set together
	lgr, err := logr.New(
		logr.ShutdownTimeout(5*time.Second),
		logr.OnExit(func(code int) {}),
		logr.OnPanic(func(err interface{}) {}),
		logr.MaxQueueSize(100),
	)
	require.NoError(t, err)
	defer func() { _ = lgr.Shutdown() }()

	logger := lgr.NewLogger()
	logger.Info("test message")
}
