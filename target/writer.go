package target

import (
	"errors"
	"io"
	"sync"

	"github.com/wiggin77/logr"
)

// Writer outputs log records to any `io.Writer`.
type Writer struct {
	Basic
	Level     logr.Level
	Fmtr      logr.Formatter
	mux       sync.Mutex
	Out       io.Writer
	MaxQueued int
}

// Start initializes the target and should start a new
// goroutine to accept incoming log records.
// In this case we just need to initialize the Basic helper
// which provides an accepting goroutine.
func (w *Writer) Start() error {
	if w.Out == nil {
		return errors.New("io.Writer cannot be nil")
	}
	w.Basic.Start(w, w, w.MaxQueued)
	return nil
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
func (w *Writer) IsLevelEnabled(level logr.Level) (enabled bool, stacktrace bool) {
	return w.Level.IsEnabled(level), w.Level.IsStacktraceEnabled(level)
}

// Formatter returns the Formatter associated with this Target.
func (w *Writer) Formatter() logr.Formatter {
	return w.Fmtr
}

// Write converts the log record to bytes, via the Formatter,
// and outputs to the io.Writer.
func (w *Writer) Write(rec *logr.LogRec) error {
	// lock to ensure we don't interleave log records.
	w.mux.Lock()
	defer w.mux.Unlock()

	data, err := w.Fmtr.Format(rec)
	if err != nil {
		return err
	}
	_, err = w.Out.Write(data)
	return err
}
