package target

import (
	"io"

	"github.com/wiggin77/logr"
)

// Writer outputs log records to any `Writer`.
type Writer struct {
	out io.Writer
}

// NewWriterTarget creates a Writer target.
func NewWriterTarget(level logr.Level, writer io.Writer) *Writer {

}

// Log outputs the log record to this targets destination.
func (w *Writer) Log(rec *logr.LogRec) {

}

// Shutdown makes best effort to flush target queue and
// frees/closes all resources.
func (w *Writer) Shutdown() error {

}
