package logr

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// LogRec collects raw, unformatted data to be logged.
type LogRec struct {
}

// Formatter turns a LogRec into a formatted string.
type Formatter interface {
}

// Target represents a destination for log records such as file,
// database, TCP socket, etc.
type Target interface {
}

// Logr provides APIs for configuration and logging.
type Logr interface {
}

// Logger implements Logr APIs.
type Logger struct {
}
