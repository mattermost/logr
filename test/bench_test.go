package test

import (
	"io/ioutil"
	"testing"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
)

// Enabled avoids compiler optimization.
var Enabled bool

// Stacktrace avoids compiler optimization.
var Stacktrace bool

// BenchmarkFilterOut benchmarks `logr.IsLevelEnabled` with empty level cache.
func BenchmarkFilterOut(b *testing.B) {
	lgr := &logr.Logr{}
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error}
		formatter := &format.Plain{Delim: " | "}
		target := target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
		lgr.AddTarget(target)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		status := lgr.IsLevelEnabled(logr.Debug)
		Enabled = status.Enabled
		Stacktrace = status.Stacktrace
	}
	b.StopTimer()
	err := lgr.Shutdown()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkLog measures adding a log record to the queue without stack trace.
// It does not measure how long the record takes to be output as that happens async.
// Level caching is enabled.
// This is how long you can expect logging to tie up the calling thread.
func BenchmarkLog(b *testing.B) {
	lgr := &logr.Logr{}
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &format.Plain{Delim: " | "}
		target := target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
		lgr.AddTarget(target)
	}

	logger := lgr.NewLogger().WithFields(logr.Fields{"name": "Wiggin"})
	logger.Errorln("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Errorf("log entry %d", b.N)
	}
	b.StopTimer()
	err := lgr.Shutdown()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkLogFiltered measures a logging call for a level that has no
// targets matching the level.  Level caching is enabled.
// This is how long you can expect logging to tie up the calling thread.
func BenchmarkLogFiltered(b *testing.B) {
	lgr := &logr.Logr{}
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Fatal}
		formatter := &format.Plain{Delim: " | "}
		target := target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
		lgr.AddTarget(target)
	}

	logger := lgr.NewLogger()
	logger.Errorln("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Log(logr.Error, "blap bleep bloop")
	}
	b.StopTimer()
	err := lgr.Shutdown()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkLogStacktrace measures adding a log record to the queue with stack trace.
// It does not measure how long the record takes to be output as that happens async.
// Level caching is enabled.
// This is how long you can expect logging to tie up the calling thread when a stack
// trace is generated.
func BenchmarkLogStacktrace(b *testing.B) {
	lgr := &logr.Logr{}
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
		formatter := &format.Plain{Delim: " | "}
		target := target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
		lgr.AddTarget(target)
	}

	logger := lgr.NewLogger()
	logger.Errorln("log entry cache primer")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Errorf("log entry with stack trace %d", b.N)
	}
	b.StopTimer()
	err := lgr.Shutdown()
	if err != nil {
		b.Error(err)
	}
}

// BenchmarkLogger measures creating Loggers with context.
func BenchmarkLogger(b *testing.B) {
	lgr := &logr.Logr{}
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Warn}
		formatter := &format.Plain{Delim: " | "}
		target := target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
		lgr.AddTarget(target)
	}

	b.ResetTimer()

	logger := lgr.NewLogger().WithFields(logr.Fields{"name": "Wiggin"})
	for i := 0; i < b.N; i++ {
		logger.Errorf("log entry %d", b.N)
	}

	b.StopTimer()
	err := lgr.Shutdown()
	if err != nil {
		b.Error(err)
	}
}
