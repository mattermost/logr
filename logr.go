package logr

import (
	"fmt"
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

// levelCacheEntry is stored in levelCache map.
type levelCacheEntry struct {
	enabled    bool
	stacktrace bool
}

var (
	logr       state
	levelCache map[string]levelCacheEntry
)

// Configure creates a logger using the supplied
// configuration.
func Configure(config *cfg.Config) error {
	// TODO
	return fmt.Errorf("not implemented yet")
}

// AddTarget adds a target to the logger which will receive
// log records for outputting.
func AddTarget(target Target) {
	logr.mux.Lock()
	defer logr.mux.Unlock()

	logr.targets = append(logr.targets, target)
	if !logr.active {
		logr.active = true
		start()
	}
}

// IsLevelEnabled returns true if at least one target has the specfified
// level enabled. The result is cached
func IsLevelEnabled(level Level) bool {

}

// ResetLevelCache resets the cached results of `IsLevelEnabled`. This is
// called any time a Target is added or a target's level is changed.
func ResetLevelCache() {

}

// start selects on incoming log records until exit channel signals.
// Incoming log records are fanned out to all log targets.
func start() {

}
