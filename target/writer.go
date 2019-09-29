package target

import (
	"io"
	"sync"

	"github.com/wiggin77/logr"
)

// Writer outputs log records to any `io.Writer`.
type Writer struct {
	Basic
	filter    logr.Filter
	formatter logr.Formatter
	out       io.Writer
	mux       sync.Mutex
}

// NewWriterTarget creates a target capable of outputting log records to an io.Writer.
func NewWriterTarget(filter logr.Filter, formatter logr.Formatter, out io.Writer, maxQueue int) *Writer {
	w := &Writer{filter: filter, formatter: formatter, out: out}
	w.Basic.Start(w, w, maxQueue)
	return w
}

// Shutdown makes best effort to flush target queue and
// frees/closes all resources.
func (w *Writer) Shutdown() error {
	w.Basic.Shutdown()
	return nil
}

// IsLevelEnabled returns true if this target should emit
// logs for the specified level. Also determines if
// a stack trace is required.
func (w *Writer) IsLevelEnabled(lvl logr.Level) (enabled bool, stacktrace bool) {
	return w.filter.IsEnabled(lvl), w.filter.IsStacktraceEnabled(lvl)
}

// Formatter returns the Formatter associated with this Target.
func (w *Writer) Formatter() logr.Formatter {
	return w.formatter
}

// Write converts the log record to bytes, via the Formatter,
// and outputs to the io.Writer.
func (w *Writer) Write(rec *logr.LogRec) error {
	// lock to ensure we don't interleave log records.
	w.mux.Lock()
	defer w.mux.Unlock()

	data, err := w.formatter.Format(rec)
	if err != nil {
		return err
	}
	_, err = w.out.Write(data)
	return err
}
