package logr

import (
	"fmt"
	"os"
	"sync"

	"github.com/wiggin77/cfg"
	"github.com/wiggin77/merror"
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// levelCacheEntry is stored in levelCache map.
type levelCacheEntry struct {
	enabled    bool
	stacktrace bool
}

// Logr maintains a list of log targets and accepts incoming
// log records.
type Logr struct {
	mux        sync.RWMutex
	targets    []Target
	active     bool
	in         chan *LogRec
	exit       chan struct{}
	shutdown   bool
	levelCache sync.Map
}

var (
	logr = &Logr{
		in:   make(chan *LogRec, MAXQUEUE),
		exit: make(chan struct{}, 1),
	}
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
		go start()
	}
}

// IsLevelEnabled returns true if at least one target has the specified
// level enabled. The result is cached so that subsequent checks are fast.
func IsLevelEnabled(level Level) (enabled bool, stacktrace bool) {
	// Don't accept new log records after shutdown.
	if logr.shutdown {
		return false, false
	}
	// Check cache.
	lce, ok := logr.levelCache.Load(level)
	if ok {
		entry := lce.(levelCacheEntry)
		return entry.enabled, entry.stacktrace
	}
	// Check each target.
	logr.mux.RLock()
	defer logr.mux.RUnlock()
	for _, t := range logr.targets {
		e, s := t.IsLevelEnabled(level)
		if e {
			enabled = true
			if s {
				stacktrace = true
			}
		}
	}
	// Cache and return the result.
	logr.levelCache.Store(level, levelCacheEntry{enabled: enabled, stacktrace: stacktrace})
	return enabled, stacktrace
}

// ResetLevelCache resets the cached results of `IsLevelEnabled`. This is
// called any time a Target is added or a target's level is changed.
func ResetLevelCache() {
	// Write lock so that new cache entries cannot be stored while we
	// clear the cache.
	logr.mux.Lock()
	defer logr.mux.Unlock()

	logr.levelCache.Range(func(key interface{}, value interface{}) bool {
		logr.levelCache.Delete(key)
		return true
	})
}

// Exit cleanly shuts down the logging engine and exits
// the process with code.
func Exit(code int) {
	Shutdown()
	os.Exit(code)
}

// Shutdown cleanly stops the logging engine after making best efforts
// to flush all targets.
func Shutdown() error {
	logr.mux.Lock()
	defer logr.mux.Unlock()

	logr.shutdown = true
	errs := merror.New()

	logr.exit <- struct{}{}

	// logr.in channel should now be drained to targets and no more log records
	// can be added.
	for _, t := range logr.targets {
		err := t.Shutdown()
		if err != nil {
			errs.Append(err)
		}
	}
	return errs.ErrorOrNil()
}

// start selects on incoming log records until exit channel signals.
// Incoming log records are fanned out to all log targets.
func start() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, r)
			go start()
		}
	}()

	for {
		var rec *LogRec
		// drain until no log records left in channel
		select {
		case rec = <-logr.in:
			rec.prep()
			fanout(rec)
		default:
		}

		// wait for log record or exit
		select {
		case rec = <-logr.in:
			rec.prep()
			fanout(rec)
		case <-logr.exit:
			return
		}
	}
}

// fanout pushes a LogRec to all targets.
func fanout(rec *LogRec) {
	var target Target
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "fanout failed for target %s, %v", target, r)
		}
	}()

	logr.mux.RLock()
	defer logr.mux.RUnlock()
	for _, target = range logr.targets {
		target.Log(rec)
	}
}
