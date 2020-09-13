package test

import (
	"io"
	"sync"
	"time"

	"github.com/mattermost/logr/v2"
)

// SlowTarget outputs log records to any `io.Writer` with configurable delay
// to simulate slower targets.
// Modify SlowTarget.Delay to determine the pause per log record.
type SlowTarget struct {
	out   io.Writer
	Delay time.Duration
	mux   sync.Mutex
}

// NewSlowTarget creates a new SlowTarget.
func NewSlowTarget(out io.Writer, delayMillis int64) *SlowTarget {
	return &SlowTarget{
		out:   out,
		Delay: time.Millisecond * time.Duration(delayMillis),
	}
}

func (st *SlowTarget) Init() error {
	return nil
}

// Write after a delay.
func (st *SlowTarget) Write(p []byte, rec *logr.LogRec) (int, error) {
	time.Sleep(st.Delay)

	st.mux.Lock()
	defer st.mux.Unlock()

	return st.out.Write(p)
}

func (st *SlowTarget) Shutdown() error {
	return nil
}
