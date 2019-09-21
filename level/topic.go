package level

import (
	"sync"
)

//
//  Just make all filtering based on topics?  Remove levels?
//
//

// Topic represents a log level based on a topic string.
type Topic string

type Topics struct {
	mux    sync.RWMutex
	topics map[Topic]bool
}

// IsEnabled returns true if the specifed Level has a topic matching
// this list of topics.
func (t *Topics) IsEnabled(l Level) bool {
	topic, ok := l.(Topic)
	if !ok {
		return false
	}
	t.mux.RLock()
	defer t.mux.RUnlock()
	_, ok = t.topics[topic]
	return ok
}

// IsStacktraceEnabled returns true if the specifed Level requires a stack trace.
func (t *Topics) IsStacktraceEnabled(l Level) bool {
	topic, ok := l.(Topic)
	if !ok {
		return false
	}
	t.mux.RLock()
	defer t.mux.RUnlock()
	stackTrace, _ := t.topics[topic]
	return stackTrace
}

// String returns a string representation of this Level.
func (t *Topics) String() string {
	switch level {
	case Panic:
		return "panic"
	case Fatal:
		return "fatal"
	case Error:
		return "error"
	case Warn:
		return "warn "
	case Info:
		return "info "
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	}
	return "unknown"
}

const (
	// Panic is the highest level of severity. Logs the message and then panics.
	Panic Std = iota
	// Fatal designates a catastrophic error. Logs the message and then calls
	// `logr.Exit(1)`.
	Fatal
	// Error designates a serious but possibly recoverable error.
	Error
	// Warn designates non-critical error.
	Warn
	// Info designates information regarding application events.
	Info
	// Debug designates verbose information typically used for debugging.
	Debug
	// Trace designates the highest verbosity of log output.
	Trace
)
