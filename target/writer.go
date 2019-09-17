package target

import (
	"io"

	"github.com/wiggin77/logr"
)

// Writer outputs log records to any `io.Writer`.
type Writer struct {
	Basic
	Out       io.Writer
	MaxQueued int
}

func (w *Writer) Start() error {

}

// NewWriterTarget creates a Writer target.
func NewWriterTarget(level logr.Level, formatter logr.Formatter, w io.Writer, maxQueued int) *Writer {
	target := &Writer{}
	target.InitBasic(level, formatter, target, maxQueued)
}

func (w *Writer) Write(rec *logr.LogRec) {

}

// Shutdown makes best effort to flush target queue and
// frees/closes all resources.
func (w *Writer) Shutdown() error {

}
