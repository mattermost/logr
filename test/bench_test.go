package test

import (
	"io"
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
		target := targets.NewWriterTarget(io.Discard)
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
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "Wiggin"))
	logger.Error("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("log entry", logr.Int("num", b.N))
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
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger()
	logger.Error("log entry cache primer")

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
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger()
	logger.Error("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("log entry with stack trace", logr.Int("num", b.N))
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
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "Wiggin"))
	//logger := lgr.NewLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Error("log entry", logr.Int("num", b.N))
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkIsLevelEnabled_SyncMap benchmarks top-level cache with syncMapLevelCache (default).
func BenchmarkIsLevelEnabled_SyncMap(b *testing.B) {
	lgr, _ := logr.New() // Uses syncMapLevelCache by default
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	// Prime the cache
	lgr.IsLevelEnabled(logr.Error)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := lgr.IsLevelEnabled(logr.Error)
		Enabled = status.Enabled
		Stacktrace = status.Stacktrace
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkIsLevelEnabled_Array benchmarks top-level cache with arrayLevelCache (legacy).
func BenchmarkIsLevelEnabled_Array(b *testing.B) {
	lgr, _ := logr.New(logr.UseArrayLevelCache(true))
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	// Prime the cache
	lgr.IsLevelEnabled(logr.Error)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := lgr.IsLevelEnabled(logr.Error)
		Enabled = status.Enabled
		Stacktrace = status.Stacktrace
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogM_3Tags_3Targets benchmarks multi-tag logging with 3 tags and 3 targets.
func BenchmarkLogM_3Tags_3Targets(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 3; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	// Create custom levels to simulate tags
	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}
	tag3 := logr.Level{ID: 102, Name: "tag3"}

	logger := lgr.NewLogger()

	// Prime the caches
	logger.LogM([]logr.Level{tag1, tag2, tag3}, "cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogM([]logr.Level{tag1, tag2, tag3}, "log entry with 3 tags")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogM_4Tags_4Targets benchmarks multi-tag logging with 4 tags and 4 targets (typical case).
func BenchmarkLogM_4Tags_4Targets(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 4; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	// Create custom levels to simulate tags
	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}
	tag3 := logr.Level{ID: 102, Name: "tag3"}
	tag4 := logr.Level{ID: 103, Name: "tag4"}

	logger := lgr.NewLogger()

	// Prime the caches
	logger.LogM([]logr.Level{tag1, tag2, tag3, tag4}, "cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogM([]logr.Level{tag1, tag2, tag3, tag4}, "log entry with 4 tags")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogM_5Tags_5Targets benchmarks multi-tag logging with 5 tags and 5 targets (worst case).
func BenchmarkLogM_5Tags_5Targets(b *testing.B) {
	lgr, _ := logr.New()
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	// Create custom levels to simulate tags
	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}
	tag3 := logr.Level{ID: 102, Name: "tag3"}
	tag4 := logr.Level{ID: 103, Name: "tag4"}
	tag5 := logr.Level{ID: 104, Name: "tag5"}

	logger := lgr.NewLogger()

	// Prime the caches
	logger.LogM([]logr.Level{tag1, tag2, tag3, tag4, tag5}, "cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogM([]logr.Level{tag1, tag2, tag3, tag4, tag5}, "log entry with 5 tags")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLogM_4Tags_4Targets_Array benchmarks multi-tag logging with arrayLevelCache (legacy).
func BenchmarkLogM_4Tags_4Targets_Array(b *testing.B) {
	lgr, _ := logr.New(logr.UseArrayLevelCache(true))
	for i := 0; i < 4; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	// Create custom levels to simulate tags
	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}
	tag3 := logr.Level{ID: 102, Name: "tag3"}
	tag4 := logr.Level{ID: 103, Name: "tag4"}

	logger := lgr.NewLogger()

	// Prime the caches
	logger.LogM([]logr.Level{tag1, tag2, tag3, tag4}, "cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogM([]logr.Level{tag1, tag2, tag3, tag4}, "log entry with 4 tags")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}

// BenchmarkLog_CustomFilter benchmarks per-target filtering with CustomFilter.
func BenchmarkLog_CustomFilter(b *testing.B) {
	lgr, _ := logr.New()

	// Create custom levels
	customLevel := logr.Level{ID: 100, Name: "custom"}

	for i := 0; i < 4; i++ {
		filter := &logr.CustomFilter{}
		filter.Add(customLevel)
		formatter := &formatters.Plain{Delim: " | "}
		target := targets.NewWriterTarget(io.Discard)
		err := lgr.AddTarget(target, "test"+strconv.Itoa(i), filter, formatter, 1000)
		require.NoError(b, err)
	}

	logger := lgr.NewLogger()

	// Prime the caches
	logger.Log(customLevel, "cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(customLevel, "log entry with custom filter")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	require.NoError(b, err)
}
