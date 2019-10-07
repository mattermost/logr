package logr

// Formatter turns a LogRec into a formatted string.
type Formatter interface {
	// Format converts a log record to bytes.
	Format(rec *LogRec, stacktrace bool) ([]byte, error)
}
