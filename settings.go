package logr

// Overridable settings and defaults.
var (
	// MaxStackFrames is the max number of stack frames collected
	// when generating stack traces for logging.
	MaxStackFrames = 30
)

// Hard settings
const (
	// DefMaxQueue determines the maximum size of the queue (channel)
	// receiving log records before forwarding to targets.
	MAXQUEUE = 1000
)
