package test

import (
	"io"
	"time"

	"github.com/mattermost/logr"
)

// SlowTarget outputs log records to any `io.Writer` with configurable delay
// to simulate slower targets.
// Modify SlowTarget.Delay to determine the pause per log record.
type SlowTarget struct {
	logr.Basic
	out   io.Writer
	Delay time.Duration
}

// NewSlowTarget creates a new SlowTarget.
func NewSlowTarget(filter logr.Filter, formatter logr.Formatter, out io.Writer, maxQueue int) *SlowTarget {
	w := &SlowTarget{out: out}
	w.Basic.Start(w, w, filter, formatter, maxQueue)
	w.Delay = time.Millisecond * 10
	return w
}

// Write converts the log record to bytes, via the Formatter,
// and outputs to the io.Writer.
func (st *SlowTarget) Write(rec *logr.LogRec) error {
	_, stacktrace := st.IsLevelEnabled(rec.Level())

	buf := rec.Logger().Logr().BorrowBuffer()
	defer rec.Logger().Logr().ReleaseBuffer(buf)

	buf, err := st.Formatter().Format(rec, stacktrace, buf)
	if err != nil {
		return err
	}

	time.Sleep(st.Delay)

	_, err = st.out.Write(buf.Bytes())
	return err
}

// String returns a string representation of this target.
func (st *SlowTarget) String() string {
	return "SlowTarget"
}
