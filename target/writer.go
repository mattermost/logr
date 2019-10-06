package target

import (
	"io"
	"sync"

	"github.com/wiggin77/logr"
)

// Writer outputs log records to any `io.Writer`.
type Writer struct {
	Basic
	out io.Writer
	mux sync.Mutex
}

// NewWriterTarget creates a target capable of outputting log records to an io.Writer.
func NewWriterTarget(filter logr.Filter, formatter logr.Formatter, out io.Writer, maxQueue int) *Writer {
	w := &Writer{out: out}
	w.Basic.Start(w, w, filter, formatter, maxQueue)
	return w
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
