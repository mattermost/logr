package logr

// Formatter turns a LogRec into a formatted string.
type Formatter interface {
	Format(rec *LogRec) ([]byte, error)
}
