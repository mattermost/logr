package logr

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/wiggin77/cfg"
	"github.com/wiggin77/merror"
)

// LevelStatus represents whether a level is enabled and
// requires a stack trace.
type LevelStatus struct {
	Enabled    bool
	Stacktrace bool
}

// Logr maintains a list of log targets and accepts incoming
// log records.
type Logr struct {
	mux                sync.RWMutex
	targets            []Target
	maxQueueSizeActual int
	in                 chan *LogRec
	done               chan struct{}
	once               sync.Once
	shutdown           bool
	levelCache         sync.Map

	// MaxQueueSize is the maximum number of log records that can be queued.
	// If exceeded, `OnQueueFull` is called which determines if the log
	// record will be dropped or block until add is successful.
	// If this is modified, it must be done before `Configure` or
	// `AddTarget`.  Defaults to DefaultMaxQueueSize.
	MaxQueueSize int

	// OnLoggerError, when not nil, is called any time an internal
	// logging error occurs. For example, this can happen when a
	// target cannot connect to its data sink.
	OnLoggerError func(error)

	// OnQueueFull, when not nil, is called on an attempt to add
	// a log record to a full Logr queue.
	// `MaxQueueSize` can be used to modify the maximum queue size.
	// This function should return quickly, with a bool indicating whether
	// the log record should be dropped (true) or block until the log record
	// is successfully added (false). If nil then blocking (false) is assumed.
	OnQueueFull func(rec *LogRec, maxQueueSize int) bool

	// OnTargetQueueFull, when not nil, is called on an attempt to add
	// a log record to a full target queue provided the target supports reporting
	// this condition.
	// This function should return quickly, with a bool indicating whether
	// the log record should be dropped (true) or block until the log record
	// is successfully added (false). If nil then blocking (false) is assumed.
	OnTargetQueueFull func(target Target, rec *LogRec, maxQueueSize int) bool

	// OnExit, when not nil, is called when a FatalXXX style log API is called.
	// When nil, then the default behavior is to cleanly shut down this Logr and
	// call `os.Exit(code)`.
	OnExit func(code int)

	// OnPanic, when not nil, is called when a PanicXXX style log API is called.
	// When nil, then the default behavior is to cleanly shut down this Logr and
	// call `panic(err)`.
	OnPanic func(err interface{})
}

// Configure adds/removes targets via the supplied `Config`.
func (logr *Logr) Configure(config *cfg.Config) error {
	// TODO
	return fmt.Errorf("not implemented yet")
}

// AddTarget adds a target to the logger which will receive
// log records for outputting.
func (logr *Logr) AddTarget(target Target) error {
	logr.mux.Lock()
	defer logr.mux.Unlock()

	if logr.shutdown {
		return fmt.Errorf("logr shut down")
	}

	logr.targets = append(logr.targets, target)

	logr.once.Do(func() {
		logr.maxQueueSizeActual = logr.MaxQueueSize
		if logr.maxQueueSizeActual == 0 {
			logr.maxQueueSizeActual = DefaultMaxQueueSize
		}
		logr.in = make(chan *LogRec, logr.maxQueueSizeActual)
		logr.done = make(chan struct{})
		go logr.start()
	})
	logr.resetLevelCache()
	return nil
}

// NewLogger creates a Logger using defaults. A `Logger` is light-weight
// enough to create on-demand, but typically one or more Loggers are
// created and re-used.
func (logr *Logr) NewLogger() *Logger {
	logger := &Logger{logr: logr}
	return logger
}

// IsLevelEnabled returns true if at least one target has the specified
// level enabled. The result is cached so that subsequent checks are fast.
func (logr *Logr) IsLevelEnabled(lvl Level) LevelStatus {
	// Check cache.
	lce, ok := logr.levelCache.Load(lvl)
	if ok {
		return lce.(LevelStatus)
	}

	logr.mux.RLock()
	defer logr.mux.RUnlock()

	status := LevelStatus{}

	// Don't accept new log records after shutdown.
	if logr.shutdown {
		return status
	}

	// Check each target.
	for _, t := range logr.targets {
		e, s := t.IsLevelEnabled(lvl)
		if e {
			status.Enabled = true
			if s {
				status.Stacktrace = true
				break // if both enabled then no sense checking more targets
			}
		}
	}

	// Cache and return the result.
	logr.levelCache.Store(lvl, status)
	return status
}

// ResetLevelCache resets the cached results of `IsLevelEnabled`. This is
// called any time a Target is added or a target's level is changed.
func (logr *Logr) ResetLevelCache() {
	// Write lock so that new cache entries cannot be stored while we
	// clear the cache.
	logr.mux.Lock()
	defer logr.mux.Unlock()
	logr.resetLevelCache()
}

// resetLevelCache empties the level cache without locking.
// mux.Lock must be held before calling this function.
func (logr *Logr) resetLevelCache() {
	logr.levelCache.Range(func(key interface{}, value interface{}) bool {
		logr.levelCache.Delete(key)
		return true
	})
}

// Enqueue adds a log record to the logr queue. If the queue is full then
// this function either blocks or the log record is dropped, depending on
// the result of calling `OnQueueFull`.
func (logr *Logr) Enqueue(rec *LogRec) {
	if logr.in == nil {
		logr.ReportError(fmt.Errorf("AddTarget or Configure must be called before Enqueue"))
	}

	select {
	case logr.in <- rec:
	default:
		if logr.OnQueueFull != nil && logr.OnQueueFull(rec, logr.maxQueueSizeActual) {
			return // drop the record
		}
		logr.in <- rec // block until success
	}
}

// exit is called by one of the FatalXXX style APIS. If `logr.OnExit` is not nil
// then that method is called, otherwise the default behavior is to shut down this
// Logr cleanly then call `os.Exit(code)`.
func (logr *Logr) exit(code int) {
	if logr.OnExit != nil {
		logr.OnExit(code)
		return
	}

	if err := logr.Shutdown(); err != nil {
		logr.ReportError(err)
	}
	os.Exit(code)
}

// panic is called by one of the PanicXXX style APIS. If `logr.OnPanic` is not nil
// then that method is called, otherwise the default behavior is to shut down this
// Logr cleanly then call `panic(err)`.
func (logr *Logr) panic(err interface{}) {
	if logr.OnPanic != nil {
		logr.OnPanic(err)
		return
	}

	if err := logr.Shutdown(); err != nil {
		logr.ReportError(err)
	}
	panic(err)
}

// Shutdown cleanly stops the logging engine after making best efforts
// to flush all targets. Call this function right before application
// exit - logr cannot be restarted once shut down.
func (logr *Logr) Shutdown() error {
	logr.mux.Lock()
	if logr.shutdown {
		return errors.New("Shutdown called again after shut down")
	}
	logr.shutdown = true
	logr.resetLevelCache()
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
	return errs.ErrorOrNil()
}

// ReportError is used to notify the host application of any internal logging errors.
// If `OnLoggerError` is not nil, it is called with the error, otherwise the error is
// output to `os.Stderr`.
func (logr *Logr) ReportError(err interface{}) {
	if logr.OnLoggerError == nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	logr.OnLoggerError(fmt.Errorf("%v", err))
}

// start selects on incoming log records until done channel signals.
// Incoming log records are fanned out to all log targets.
func (logr *Logr) start() {
	defer func() {
		if r := recover(); r != nil {
			logr.ReportError(r)
			go logr.start()
		}
	}()

	var rec *LogRec
	var more bool
	for {
		select {
		case rec, more = <-logr.in:
			if more {
				rec.prep()
				logr.fanout(rec)
			} else {
				close(logr.done)
				return
			}
		}
	}
}

// fanout pushes a LogRec to all targets.
func (logr *Logr) fanout(rec *LogRec) {
	var target Target
	defer func() {
		if r := recover(); r != nil {
			logr.ReportError(fmt.Errorf("fanout failed for target %s, %v", target, r))
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
