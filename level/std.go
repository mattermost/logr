package level

// Std represents the classic log levels provided by Stdlib, Logrus and others.
type Std uint32

// StdStacktraceLevel determines the level at which stack traces are needed.
// The default is Fatal, meaning any log record at severity fatal (or greater severity)
// will have a stack trace generated.
var StdStacktraceLevel = Fatal

// IsEnabled returns true if the specifed Level is at or above this verbosity. Also
// determines if a stack trace is required.
func (level Std) IsEnabled(l Level) bool {
	lvl, ok := l.(Std)
	if !ok {
		return false
	}
	return lvl <= level
}

// IsStacktraceEnabled returns true if the specifed Level requires a stack trace.
func (level Std) IsStacktraceEnabled(l Level) bool {
	lvl, ok := l.(Std)
	if !ok {
		return false
	}
	return lvl <= StdStacktraceLevel
}

// String returns a string representation of this Level.
func (level Std) String() string {
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
