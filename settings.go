package logr

// Overridable settings and defaults.
var (
	// MaxQueueSize is the maximum number of log records that can be queued.
	// If exceeded, `OnQueueFull` is called which determines if the log
	// record will be dropped or block until add is successful.
	// If this is modified, it must be done before `logr.Configure` or
	// `logr.AddTarget`.
	MaxQueueSize = 1000

	// MaxStackFrames is the max number of stack frames collected
	// when generating stack traces for logging.
	MaxStackFrames = 30
)
