package target

import (
	"fmt"
	"os"

	"github.com/wiggin77/logr"
)

// RecordWriter can convert a LogRecord to bytes and output to some data sink.
type RecordWriter interface {
	Write(rec *logr.LogRec) error
}

// Basic provides the basic functionality of a Target that can be used
// to more easily compose your own Targets.
type Basic struct {
	target logr.Target
	in     chan *logr.LogRec
	done   chan struct{}
	w      RecordWriter
}

// Start initializes this target helper and starts accepting log records for processing.
func (b *Basic) Start(target logr.Target, rw RecordWriter, maxQueued int) {
	b.target = target
	b.in = make(chan *logr.LogRec, maxQueued)
	b.done = make(chan struct{}, 1)
	b.w = rw
	go b.start()
}

// Shutdown stops processing log records after making best
// effort to flush queue.
func (b *Basic) Shutdown() error {
	b.done <- struct{}{}
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
	for {
		var rec *logr.LogRec
		// drain until no log records left in channel
		select {
		case rec = <-b.in:
			err = b.w.Write(rec)
			if err != nil {
				rec.Logger().Logr().ReportError(err)
			}
		default:
		}

		// wait for log record or exit
		select {
		case rec = <-b.in:
			err = b.w.Write(rec)
			if err != nil {
				rec.Logger().Logr().ReportError(err)
			}
		case <-b.done:
			return
		}
	}
}
