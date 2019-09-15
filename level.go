package logr

import (
	"fmt"
)

// Level provides a mechanism to enable/disable specific log lines.
// A default implementation using "panic, fatal..." is provided, however
// more flexible implementations are possible such as topic names.
// TODO: create example using topic strings.
type Level interface {
	fmt.Stringer
	IsEnabled(Level) bool
}

// StdLevel represents the classic log levels provided by Stdlib, Logrus and others.
type StdLevel uint32

// IsEnabled returns true if the specifed Level is at or above this verbosity.
func (level StdLevel) IsEnabled(l Level) bool {
	lvl, ok := l.(StdLevel)
	if !ok {
		return false
	}
	return lvl >= level
}

// String returns a string representation of this Level.
func (level StdLevel) String() string {
	switch level {
	case PanicLevel:
		return "panic"
	case FatalLevel:
		return "fatal"
	case ErrorLevel:
		return "error"
	case WarnLevel:
		return "warn"
	case InfoLevel:
		return "info"
	case DebugLevel:
		return "debug"
	case TraceLevel:
		return "trace"
	}
	return "unknown"
}

const (
	// PanicLevel is the highest level of severity and the least verbose.
	// Logs the message and then panics.
	PanicLevel StdLevel = iota
	// FatalLevel designates a catastrophic error. Logs the message and then calls
	// `logger.Exit(1)`.
	FatalLevel
	// ErrorLevel designates a serious but recoverable error.
	ErrorLevel
	// WarnLevel designates non-critical error.
	WarnLevel
	// InfoLevel designates information regarding application events.
	InfoLevel
	// DebugLevel designates verbose information typically used for debugging.
	DebugLevel
	// TraceLevel designates the highest verbosity of log output.
	TraceLevel
)
