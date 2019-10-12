package logr

// StdLevel represents the classic log levels provided by Stdlib, Logrus and others.
type StdLevel struct {
	id   LevelID
	name string
}

// ID returns the unique id of this Level.
func (level StdLevel) ID() LevelID {
	return level.id
}

// String returns a string representation of this Level.
func (level StdLevel) String() string {
	return level.name
}

// StdFilter allows targets to filter via classic log levels where any level
// beyond a certain verbosity/severity is enabled.
type StdFilter struct {
	Lvl        StdLevel
	Stacktrace StdLevel
}

// IsEnabled returns true if the specified Level is at or above this verbosity. Also
// determines if a stack trace is required.
func (lt StdFilter) IsEnabled(level Level) bool {
	lvl, ok := level.(StdLevel)
	if !ok {
		return false
	}
	return lvl.id <= lt.Lvl.id
}

// IsStacktraceEnabled returns true if the specified Level requires a stack trace.
func (lt StdFilter) IsStacktraceEnabled(level Level) bool {
	lvl, ok := level.(StdLevel)
	if !ok {
		return false
	}
	return lvl.id <= lt.Stacktrace.id
}

var (
	// Panic is the highest level of severity. Logs the message and then panics.
	Panic = StdLevel{id: 0, name: "panic"}
	// Fatal designates a catastrophic error. Logs the message and then calls
	// `logr.Exit(1)`.
	Fatal = StdLevel{id: 1, name: "fatal"}
	// Error designates a serious but possibly recoverable error.
	Error = StdLevel{id: 2, name: "error"}
	// Warn designates non-critical error.
	Warn = StdLevel{id: 3, name: "warn"}
	// Info designates information regarding application events.
	Info = StdLevel{id: 4, name: "info"}
	// Debug designates verbose information typically used for debugging.
	Debug = StdLevel{id: 5, name: "debug"}
	// Trace designates the highest verbosity of log output.
	Trace = StdLevel{id: 6, name: "trace"}
)
