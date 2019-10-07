package target

import (
	"fmt"
	"os"
	"time"

	"github.com/wiggin77/logr"
)

// RecordWriter can convert a LogRecord to bytes and output to some data sink.
type RecordWriter interface {
	Write(rec *logr.LogRec) error
}

// Basic provides the basic functionality of a Target that can be used
// to more easily compose your own Targets. To use, just embed Basic
// in your target type, implement `RecordWriter`, and call `Start`.
type Basic struct {
	target logr.Target

	filter    logr.Filter
	formatter logr.Formatter

	in   chan *logr.LogRec
	done chan struct{}
	w    RecordWriter
}

// Start initializes this target helper and starts accepting log records for processing.
func (b *Basic) Start(target logr.Target, rw RecordWriter, filter logr.Filter, formatter logr.Formatter, maxQueued int) {
	b.target = target
	b.filter = filter
	b.formatter = formatter
	b.in = make(chan *logr.LogRec, maxQueued)
	b.done = make(chan struct{}, 1)
	b.w = rw
	go b.start()
}

// IsLevelEnabled returns true if this target should emit
// logs for the specified level. Also determines if
// a stack trace is required.
func (b *Basic) IsLevelEnabled(lvl logr.Level) (enabled bool, stacktrace bool) {
	return b.filter.IsEnabled(lvl), b.filter.IsStacktraceEnabled(lvl)
}

// Formatter returns the Formatter associated with this Target.
func (b *Basic) Formatter() logr.Formatter {
	return b.formatter
}

// Shutdown stops processing log records after making best
// effort to flush queue.
func (b *Basic) Shutdown() error {
	// close the incoming channel and wait for read loop to exit.
	close(b.in)
	select {
	case <-time.After(time.Second * 10):
	case <-b.done:
	}

	// b.in channel should now be drained.
	return nil
}

// Log outputs the log record to this targets destination.
func (b *Basic) Log(rec *logr.LogRec) {
	select {
	case b.in <- rec:
	default:
		handler := rec.Logger().Logr().OnTargetQueueFull
		if handler != nil && handler(b.target, rec, cap(b.in)) {
			return // drop the record
		}
		b.in <- rec // block until success
	}
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

	var err error
	var rec *logr.LogRec
	var more bool
	for {
		select {
		case rec, more = <-b.in:
			if more {
				err = b.w.Write(rec)
				if err != nil {
					rec.Logger().Logr().ReportError(err)
				}
			} else {
				close(b.done)
				return
			}
		}
	}
}
