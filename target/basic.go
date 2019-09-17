package target

import (
	"fmt"
	"os"

	"github.com/wiggin77/logr"
)

// RecordWriter can convert a LogRecord to bytes and output to some data sink.
type RecordWriter interface {
	Write(rec *logr.LogRec)
}

// Basic provides the basic functionality of a Target that can be used
// to more easily compose your own Targets.
type Basic struct {
	Level logr.Level
	Fmtr  logr.Formatter
	In    chan *logr.LogRec
	Done  chan struct{}
	W     RecordWriter
}

// InitBasic creates a Basic target that can be used within custom targets.
func (b *Basic) InitBasic(level logr.Level, formatter logr.Formatter, rw RecordWriter, maxQueued int) {
	b.Level = level
	b.Fmtr = formatter
	b.In = make(chan *logr.LogRec, maxQueued)
	b.Done = make(chan struct{}, 1)
	b.W = rw
	go b.start()
}

// IsLevelEnabled returns true if this target should emit
// logs for the specified level. Also determines if
// a stack trace is required.
func (b *Basic) IsLevelEnabled(level logr.Level) (enabled bool, stacktrace bool) {
	return b.Level.IsEnabled(level), b.Level.IsStacktraceEnabled(level)
}

// Formatter returns the Formatter associated with this Target.
func (b *Basic) Formatter() logr.Formatter {
	return b.Fmtr
}

// Log outputs the log record to this targets destination.
func (b *Basic) Log(rec *logr.LogRec) {
	b.In <- rec
}

// Start accepts log records via In channel and writes to the
// supplied writer, until Done channel signaled.
func (b *Basic) start() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, r)
			go b.start()
		}
	}()

	for {
		var rec *logr.LogRec
		// drain until no log records left in channel
		select {
		case rec = <-b.In:
			W.Write(rec)
		default:
		}

		// wait for log record or exit
		select {
		case rec = <-b.In:
			W.Write(rec)
		case <-b.Done:
			return
		}
	}
}
