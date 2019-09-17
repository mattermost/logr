package target

import "github.com/wiggin77/logr"

// Basic provides the basic functionality of a Target that can be used
// to more easily compose your own Targets.
type Basic struct {
	level     logr.Level
	formatter logr.Formatter
	in        chan *logr.LogRec
	done      chan struct{}
}

// NewBasicTarget creates a Basic target that can be used within custom targets.
func NewBasicTarget(level logr.Level, formatter logr.Formatter, maxQueued int) *Basic {
	return &Basic{
		level:     level,
		formatter: formatter,
		in:        make(chan *logr.LogRec, maxQueued),
		done:      make(chan struct{}),
	}
}

// IsLevelEnabled returns true if this target should emit
// logs for the specified level. Also determines if
// a stack trace is required.
func (b *Basic) IsLevelEnabled(level logr.Level) (enabled bool, stacktrace bool) {
	return b.level.IsEnabled(level), b.level.IsStacktraceEnabled(level)
}

// Formatter returns the Formatter associated with this Target.
func (b *Basic) Formatter() logr.Formatter {
	return b.formatter
}

// Shutdown makes best effort to flush target queue and
// frees/closes all resources.
func (b *Basic) Shutdown() error {

}

// Start accepts
func (b *Basic) Start(fn func(rec *logr.LogRec)) {

}
