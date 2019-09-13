package logr

import (
	"sync"

	"github.com/wiggin77/cfg"
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// Formatter turns a LogRec into a formatted string.
type Formatter interface {
}

// Target represents a destination for log records such as file,
// database, TCP socket, etc.
type Target interface {
}

// state maintains a list of log targets
type state struct {
	mux     sync.Mutex
	targets []Target
	active  bool
	in      chan LogRec
	exit    chan struct{}
}

// Configure creates a logger using the supplied
// configuration.
func Configure(config *cfg.Config) (*Logger, error) {
	logger := &Logger{}
	err := configLogger(config)
	return logger, err
}

// AddTarget adds a target to the logger which will receive
// log records for outputting.
func AddTarget(target Target) {
	logger.mux.Lock()
	defer logger.mux.Unlock()

	logger.targets = append(logger.targets, target)
	if !logger.active {
		logger.active = true
		logger.start()
	}
}

// start selects on incoming log records until exit channel signals.
// Incoming log records are fanned out to all log targets.
func start() {

}
