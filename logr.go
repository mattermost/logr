package logr

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/wiggin77/cfg"
	"github.com/wiggin77/merror"
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// LevelStatus represents whether a level is enabled and
// requires a stack trace.
type LevelStatus struct {
	Enabled    bool
	Stacktrace bool
}

// Logr maintains a list of log targets and accepts incoming
// log records.
type Logr struct {
	mux          sync.RWMutex
	targets      []Target
	active       bool
	in           chan *LogRec
	done         chan struct{}
	shuttingdown bool
	levelCache   sync.Map
}

var (
	logr = &Logr{}

	// OnLoggerError when not nil, is called any time an internal
	// logging error occurs. For example, this can happen when a
	// target cannot connect to its data sink.
	OnLoggerError func(error)

	// OnQueueFull when not nil, is called on an attempt to add
	// a log record to a full Logr queue.
	// `MaxQueueSize` can be used to modify the maximum queue size.
	// This function should return quickly, with a bool indicating whether
	// the log record should be dropped (true) or block until the log record
	// is successfully added (false). If nil then blocking (false) is assumed.
	OnQueueFull func(rec *LogRec, maxQueueSize int) bool

	// OnTargetQueueFull when not nil, is called on an attempt to add
	// a log record to a full target queue.
	// This function should return quickly, with a bool indicating whether
	// the log record should be dropped (true) or block until the log record
	// is successully added (false). If nil then blocking (false) is assumed.
	OnTargetQueueFull func(target Target, rec *LogRec, maxQueueSize int) bool
)

// Configure adds/removes targets via the supplied `Config`.
func Configure(config *cfg.Config) error {
	// TODO
	return fmt.Errorf("not implemented yet")
}

// AddTarget adds a target to the logger which will receive
// log records for outputting.
func AddTarget(target Target) error {
	logr.mux.Lock()
	defer logr.mux.Unlock()

	if logr.shuttingdown {
		return fmt.Errorf("logr shutting down")
	}

	logr.targets = append(logr.targets, target)
	if !logr.active {
		logr.active = true
		go start()
	}
	resetLevelCache()
	return nil
}

// IsLevelEnabled returns true if at least one target has the specified
// level enabled. The result is cached so that subsequent checks are fast.
func IsLevelEnabled(level Level) LevelStatus {
	// Check cache.
	lce, ok := logr.levelCache.Load(level)
	if ok {
		return lce.(LevelStatus)
	}

	logr.mux.RLock()
	defer logr.mux.RUnlock()

	status := LevelStatus{}

	// Don't accept new log records while shutting down.
	if logr.shuttingdown {
		return status
	}

	// Check each target.
	for _, t := range logr.targets {
		e, s := t.IsLevelEnabled(level)
		if e {
			status.Enabled = true
			if s {
				status.Stacktrace = true
				break // if both enabled then no sense checking more targets
			}
		}
	}

	// Cache and return the result.
	logr.levelCache.Store(level, status)
	return status
}

// ResetLevelCache resets the cached results of `IsLevelEnabled`. This is
// called any time a Target is added or a target's level is changed.
func ResetLevelCache() {
	// Write lock so that new cache entries cannot be stored while we
	// clear the cache.
	logr.mux.Lock()
	defer logr.mux.Unlock()
	resetLevelCache()
}

// resetLevelCache empties the level cache without locking.
// mux.Lock must be held before calling this function.
func resetLevelCache() {
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
	logr.shuttingdown = true
	resetLevelCache()
	logr.mux.Unlock()

	// close the incoming channel and wait for read loop to exit.
	close(logr.in)
	select {
	case <-time.After(time.Second * 10):
	case <-logr.done:
	}

	// logr.in channel should now be drained to targets and no more log records
	// can be added.
	logr.mux.Lock()
	defer logr.mux.Unlock()
	errs := merror.New()
	for _, t := range logr.targets {
		err := t.Shutdown()
		if err != nil {
			errs.Append(err)
		}
	}

	// reset logr so it can be restarted by adding new targets.
	logr.targets = nil
	logr.in = nil
	logr.done = nil
	logr.shuttingdown = false
	return errs.ErrorOrNil()
}

// ReportError is used to notify the host application of any internal logging errors.
// If `OnLoggerError` is not nil, it is called with the error, otherwise the error is
// output to `os.Stderr`.
func ReportError(err interface{}) {
	if OnLoggerError == nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	OnLoggerError(fmt.Errorf("%v", err))
}

// start selects on incoming log records until done channel signals.
// Incoming log records are fanned out to all log targets.
func start() {
	defer func() {
		if r := recover(); r != nil {
			ReportError(r)
			go start()
		} else {
			logr.mux.Lock()
			logr.active = false
			logr.mux.Unlock()
		}
	}()

	logr.in = make(chan *LogRec, MaxQueueSize)
	logr.done = make(chan struct{})

	for {
		var rec *LogRec
		var more bool
		select {
		case rec, more = <-logr.in:
			if more {
				rec.prep()
				fanout(rec)
			} else {
				close(logr.done)
				return
			}
		}
	}
}

// fanout pushes a LogRec to all targets.
func fanout(rec *LogRec) {
	var target Target
	defer func() {
		if r := recover(); r != nil {
			ReportError(fmt.Errorf("fanout failed for target %s, %v", target, r))
		}
	}()

	logr.mux.RLock()
	defer logr.mux.RUnlock()
	for _, target = range logr.targets {
		if enabled, _ := target.IsLevelEnabled(rec.Level()); enabled {
			target.Log(rec)
		}
	}
}
