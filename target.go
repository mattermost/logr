package logr

// Target represents a destination for log records such as file,
// database, TCP socket, etc.
type Target interface {
	// IsLevelEnabled returns true if this target should emit
	// logs for the specified level. Also determines if
	// a stack trace is required.
	IsLevelEnabled(Level) (enabled bool, stacktrace bool)

	// Log outputs the log record to this targets destination.
	Log(rec *LogRec)

	// Shutdown makes best effort to flush target queue and
	// frees/closes all resources.
	Shutdown() error
}
