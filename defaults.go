package logr

var (
	// DEF_MAX_QUEUE determines the maximum size of the queue
	// receiving logs before forwarding to targets. Changing
	// this affects all Loggers created thereafter.
	defMaxQueue = 100
)
