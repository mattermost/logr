package logr

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/wiggin77/logr/level"
)

// Level aliases `level.Level` to avoid stutter.
type Level level.Level

// LogRec collects raw, unformatted data to be logged.
// TODO:  pool these?  how to reliably know when targets are done with them?
type LogRec struct {
	mux  sync.RWMutex
	time time.Time

	level  Level
	logger *Logger

	template string
	newline  bool
	args     []interface{}

	stackPC    []uintptr
	stackCount int

	// remaining fields calculated by `prep`
	msg    string
	frames []runtime.Frame
}

// NewLogRec creates a new LogRec with the current time and optional stack trace.
func NewLogRec(level Level, logger *Logger, template string, args []interface{}, incStacktrace bool) *LogRec {
	rec := &LogRec{time: time.Now(), logger: logger, level: level, template: template, args: args}
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

// WithTime returns a shallow copy of the log record while replacing
// the time. This can be used by targets and formatters to adjust
// the time, or take ownership of the log record.
func (rec *LogRec) WithTime(time time.Time) *LogRec {
	rec.mux.RLock()
	defer rec.mux.RUnlock()

	return &LogRec{
		time:       time,
		level:      rec.level,
		logger:     rec.logger,
		template:   rec.template,
		newline:    rec.newline,
		args:       rec.args,
		msg:        rec.msg,
		stackPC:    rec.stackPC,
		stackCount: rec.stackCount,
		frames:     rec.frames,
	}
}

// Time returns this log record's time stamp.
func (rec *LogRec) Time() time.Time {
	// no locking needed as this field is not mutated.
	return rec.time
}

// Level returns this log record's Level.
func (rec *LogRec) Level() Level {
	// no locking needed as this field is not mutated.
	return rec.level
}

// Fields returns this log record's Fields.
func (rec *LogRec) Fields() Fields {
	// no locking needed as this field is not mutated.
	return rec.logger.fields
}

// Msg returns this log record's message text.
func (rec *LogRec) Msg() string {
	rec.mux.RLock()
	defer rec.mux.RUnlock()
	return rec.msg
}

// StackFrames returns this log record's stack frames or
// nil if no stack trace was required.
func (rec *LogRec) StackFrames() []runtime.Frame {
	rec.mux.RLock()
	defer rec.mux.RUnlock()
	return rec.frames
}
