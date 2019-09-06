package logr

import "github.com/wiggin77/cfg"

// Logger implements Logr APIs.
// TODO expand docs for this key struct
type Logger struct {
	fields  Fields
	targets []Target
	in      chan LogRec
	exit chan struct{}{}
}

// NewLogger creates a logger using defaults.
func NewLogger() *Logger {
	logger := &Logger{}
	logger.in = make(chan LogRec, defMaxQueue)
	logger.exit = make(chan struct{}{}, 1)
	return logger
}

// NewLoggerFromConfig creates a logger using the supplied
// configuration.
func NewLoggerFromConfig(config *cfg.Config) (*Logger, error) {
	logger := &Logger{}
	err := configLogger(config)
	return logger, err
}

// start 
func (logger *Logger) start() {

}