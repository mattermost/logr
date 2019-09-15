package logr

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// LogRec collects raw, unformatted data to be logged.
// TODO:  pool these?  how to reliably know when targets are done with them?
type LogRec struct {
	mux  sync.RWMutex
	Time time.Time

	level  Level
	logger *Logger

	template string
	newline  bool
	args     []interface{}

	msg string

	stackPC    []uintptr
	stackCount int
	frames     []runtime.Frame
}

// NewLogRec creates a new LogRec with the current time and optional stack trace.
func NewLogRec(level Level, logger *Logger, template string, args []interface{}, incStacktrace bool) *LogRec {
	rec := &LogRec{Time: time.Now(), logger: logger, level: level, template: template, args: args}
	if incStacktrace {
		rec.stackPC = make([]uintptr, MaxStackFrames)
		rec.stackCount = runtime.Callers(2, rec.stackPC)
	}
	return rec
}

// prep resolves all args and field values to strings, and
// resolves stack trace to frames.
func (rec *LogRec) prep() {
	rec.mux.Lock()
	defer rec.mux.Unlock()

	// resolve args
	if rec.template == "" {
		if rec.newline {
			rec.msg = fmt.Sprintln(rec.args...)
		} else {
			rec.msg = fmt.Sprint(rec.args...)
		}
	} else {
		rec.msg = fmt.Sprintf(rec.template, rec.args...)
	}

	// resolve stack trace
	if rec.stackCount > 0 {
		frames := runtime.CallersFrames(rec.stackPC[:rec.stackCount])
		for {
			f, more := frames.Next()
			rec.frames = append(rec.frames, f)
			if !more {
				break
			}
		}
	}
}

// Format returns a string representation of this log record using
// the specified Formatter.
func (rec *LogRec) Format(formatter Formatter) (string, error) {
	return formatter.Format(rec)
}
