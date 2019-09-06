package logr

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

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
