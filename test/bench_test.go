package test

import (
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/require"
)

// Enabled avoids compiler optimization.
var Enabled bool

// Stacktrace avoids compiler optimization.
var Stacktrace bool

// BenchmarkFilterOut benchmarks `logr.IsLevelEnabled` with empty level cache.
func BenchmarkFilterOut(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(ioutil.Discard)
		err := lgr.AddTarget(target, "benchmarkTest", filter, formatter, 1000)
		require.NoError(b, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := lgr.IsLevelEnabled(logr.Debug)
		Enabled = status.Enabled
		Stacktrace = status.Stacktrace
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLog measures adding a log record to the queue without stack trace.
// It does not measure how long the record takes to be output as that happens async.
// Level caching is enabled.
// This is how long you can expect logging to tie up the calling thread.
func BenchmarkLog(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(ioutil.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger().WithFields(logr.Fields{"name": "Wiggin"})
	logger.Errorln("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Errorf("log entry %d", b.N)
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogFiltered measures a logging call for a level that has no
// targets matching the level.  Level caching is enabled.
// This is how long you can expect logging to tie up the calling thread.
func BenchmarkLogFiltered(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Fatal}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(ioutil.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger()
	logger.Errorln("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(logr.Error, "blap bleep bloop")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogStacktrace measures adding a log record to the queue with stack trace.
// It does not measure how long the record takes to be output as that happens async.
// Level caching is enabled.
// This is how long you can expect logging to tie up the calling thread when a stack
// trace is generated.
func BenchmarkLogStacktrace(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(ioutil.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger()
	logger.Errorln("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Errorf("log entry with stack trace %d", b.N)
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogger measures creating Loggers with context.
func BenchmarkLogger(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(ioutil.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger().WithFields(logr.Fields{"name": "Wiggin"})
	//logger := lgr.NewLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Errorf("log entry %d", b.N)
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}
