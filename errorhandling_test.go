package logr_test

import (
	"bytes"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FailingTarget is a target that fails on Init.
type FailingInitTarget struct {
	initCalled bool
}

func (f *FailingInitTarget) Init() error {
	f.initCalled = true
	return errors.New("init failed")
}

func (f *FailingInitTarget) Write(p []byte, rec *logr.LogRec) (int, error) {
	return len(p), nil
}

func (f *FailingInitTarget) Shutdown() error {
	return nil
}

// TestErrorHandling_TargetInitFailure tests AddTarget with failing Init().
func TestErrorHandling_TargetInitFailure(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	failTarget := &FailingInitTarget{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}

	err = lgr.AddTarget(failTarget, "failing", filter, formatter, 100)
	assert.Error(t, err, "AddTarget should return error when Init fails")
	assert.True(t, failTarget.initCalled, "Init should have been called")

	// Logr should still be usable with other targets
	buf := &bytes.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "working", filter, formatter, 100)
	assert.NoError(t, err)
}

// FailingWriteTarget is a target that fails on Write.
type FailingWriteTarget struct {
	writeCalls int32
	failCount  int // Fail first N writes
}

func (f *FailingWriteTarget) Init() error {
	return nil
}

func (f *FailingWriteTarget) Write(p []byte, rec *logr.LogRec) (int, error) {
	calls := atomic.AddInt32(&f.writeCalls, 1)
	if int(calls) <= f.failCount {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func (f *FailingWriteTarget) Shutdown() error {
	return nil
}

// TestErrorHandling_TargetWriteFailure tests Write failures.
func TestErrorHandling_TargetWriteFailure(t *testing.T) {
	var errorCount int32
	onError := func(err error) {
		atomic.AddInt32(&errorCount, 1)
	}

	lgr, err := logr.New(logr.OnLoggerError(onError))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	failTarget := &FailingWriteTarget{failCount: 3}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	err = lgr.AddTarget(failTarget, "failing", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Send messages - first 3 should fail
	for i := 0; i < 5; i++ {
		logger.Error("test message", logr.Int("id", i))
	}

	err = lgr.Flush()
	require.NoError(t, err)

	// Should have reported errors
	errors := atomic.LoadInt32(&errorCount)
	assert.GreaterOrEqual(t, errors, int32(3), "Should have error reports")
	assert.Equal(t, int32(5), atomic.LoadInt32(&failTarget.writeCalls), "All writes attempted")
}

// FailingShutdownTarget is a target that fails on Shutdown.
type FailingShutdownTarget struct {
	*bytes.Buffer
	shutdownCalled bool
}

func (f *FailingShutdownTarget) Init() error {
	f.Buffer = &bytes.Buffer{}
	return nil
}

func (f *FailingShutdownTarget) Write(p []byte, rec *logr.LogRec) (int, error) {
	return f.Buffer.Write(p)
}

func (f *FailingShutdownTarget) Shutdown() error {
	f.shutdownCalled = true
	return errors.New("shutdown failed")
}

// TestErrorHandling_TargetShutdownFailure tests Shutdown failures.
func TestErrorHandling_TargetShutdownFailure(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)

	failTarget := &FailingShutdownTarget{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	err = lgr.AddTarget(failTarget, "failing", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Error("test message")

	err = lgr.Shutdown()
	assert.Error(t, err, "Shutdown should return error when target shutdown fails")
	assert.True(t, failTarget.shutdownCalled, "Shutdown should have been called")
}

// TestErrorHandling_OnLoggerError tests OnLoggerError callback.
func TestErrorHandling_OnLoggerError(t *testing.T) {
	var lastError error
	var errorCount int32

	onError := func(err error) {
		lastError = err
		atomic.AddInt32(&errorCount, 1)
	}

	lgr, err := logr.New(logr.OnLoggerError(onError))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Report an error manually
	lgr.ReportError(errors.New("test error"))

	// Give time for callback
	err = lgr.Flush()
	require.NoError(t, err)

	assert.NotNil(t, lastError)
	assert.Contains(t, lastError.Error(), "test error")
	assert.Equal(t, int32(1), atomic.LoadInt32(&errorCount))
}

// TestErrorHandling_MaxLevelIDExceeded tests behavior when level ID is too large.
func TestErrorHandling_MaxLevelIDExceeded(t *testing.T) {
	var errorCount int32
	onError := func(err error) {
		atomic.AddInt32(&errorCount, 1)
	}

	lgr, err := logr.New(logr.OnLoggerError(onError))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	invalidLevel := logr.Level{ID: logr.MaxLevelID + 1, Name: "invalid"}
	filter := &logr.CustomFilter{}
	filter.Add(invalidLevel)

	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Log(invalidLevel, "this triggers error")

	err = lgr.Flush()
	require.NoError(t, err)

	errors := atomic.LoadInt32(&errorCount)
	assert.Greater(t, errors, int32(0), "Should report error for invalid level ID")
}

// TestErrorHandling_MultipleErrorsDuringOperation tests multiple concurrent errors.
func TestErrorHandling_MultipleErrorsDuringOperation(t *testing.T) {
	var errorCount int32
	onError := func(err error) {
		atomic.AddInt32(&errorCount, 1)
	}

	lgr, err := logr.New(logr.OnLoggerError(onError), logr.MaxQueueSize(100))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Add multiple failing targets
	for i := 0; i < 3; i++ {
		failTarget := &FailingWriteTarget{failCount: 10}
		filter := &logr.StdFilter{Lvl: logr.Error}
		formatter := &formatters.Plain{Delim: " | "}
		err = lgr.AddTarget(failTarget, "fail", filter, formatter, 50)
		require.NoError(t, err)
	}

	logger := lgr.NewLogger()

	// Send many messages
	for i := 0; i < 20; i++ {
		logger.Error("error test", logr.Int("id", i))
	}

	err = lgr.Flush()
	require.NoError(t, err)

	// Should have many error reports (3 targets * up to 10 failures each)
	errors := atomic.LoadInt32(&errorCount)
	assert.Greater(t, errors, int32(0), "Should report multiple errors")
	t.Logf("Reported %d errors", errors)
}

// TestErrorHandling_NoOnLoggerError tests default behavior without OnLoggerError.
func TestErrorHandling_NoOnLoggerError(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// ReportError should not panic even without callback
	lgr.ReportError(errors.New("test error"))

	// Should still work normally
	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Error("normal operation")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "normal operation")
}
