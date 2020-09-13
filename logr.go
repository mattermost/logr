package logr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wiggin77/merror"
)

// Logr maintains a list of log targets and accepts incoming
// log records.  Use `New` to create instances.
type Logr struct {
	tmux        sync.RWMutex // targetHosts mutex
	targetHosts []*TargetHost

	in         chan *LogRec
	quit       chan struct{} // closed by Shutdown to exit read loop
	done       chan struct{} // closed when read loop exited
	lvlCache   levelCache
	bufferPool sync.Pool
	options    *options
	metrics    *metrics

	shutdown int32
}

// New creates a new Logr instance with one or more options specified.
// Some options with invalid values can cause an error to be returned,
// however `logr.New()` using just defaults never errors.
func New(opts ...Option) (*Logr, error) {
	options := &options{
		maxQueueSize:    DefaultMaxQueueSize,
		enqueueTimeout:  DefaultEnqueueTimeout,
		shutdownTimeout: DefaultShutdownTimeout,
		flushTimeout:    DefaultFlushTimeout,
		maxPooledBuffer: DefaultMaxPooledBuffer,
	}

	logr := &Logr{options: options}

	// apply the options
	for _, opt := range opts {
		if err := opt(logr); err != nil {
			return nil, err
		}
	}
	_ = StackFilter(logrPkg, logrPkg+"/target", logrPkg+"/format")(logr)

	logr.in = make(chan *LogRec, logr.options.maxQueueSize)
	logr.quit = make(chan struct{})
	logr.done = make(chan struct{})

	if logr.options.useSyncMapLevelCache {
		logr.lvlCache = &syncMapLevelCache{}
	} else {
		logr.lvlCache = &arrayLevelCache{}
	}
	logr.lvlCache.setup()

	logr.bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	logr.initMetrics()

	go logr.start()

	return logr, nil
}

// AddTarget adds a target to the logger which will receive
// log records for outputting.
func (logr *Logr) AddTarget(target Target, name string, filter Filter, formatter Formatter, maxQueueSize int) error {
	if logr.IsShutdown() {
		return fmt.Errorf("AddTarget called after Logr shut down")
	}

	hostOpts := targetHostOptions{
		name:                    name,
		filter:                  filter,
		formatter:               formatter,
		maxQueueSize:            maxQueueSize,
		collector:               logr.options.metricsCollector,
		metricsUpdateFreqMillis: logr.options.metricsUpdateFreqMillis,
	}

	host, err := newTargetHost(target, hostOpts)
	if err != nil {
		return err
	}

	logr.tmux.Lock()
	defer logr.tmux.Unlock()

	logr.targetHosts = append(logr.targetHosts, host)

	logr.ResetLevelCache()

	return nil
}

// NewLogger creates a Logger using defaults. A `Logger` is light-weight
// enough to create on-demand, but typically one or more Loggers are
// created and re-used.
func (logr *Logr) NewLogger() Logger {
	logger := Logger{logr: logr}
	return logger
}

var levelStatusDisabled = LevelStatus{}

// IsLevelEnabled returns true if at least one target has the specified
// level enabled. The result is cached so that subsequent checks are fast.
func (logr *Logr) IsLevelEnabled(lvl Level) LevelStatus {
	// No levels enabled after shutdown
	if atomic.LoadInt32(&logr.shutdown) != 0 {
		return levelStatusDisabled
	}

	// Check cache.
	status, ok := logr.lvlCache.get(lvl.ID)
	if ok {
		return status
	}

	// Cache miss; check each target.
	logr.tmux.RLock()
	defer logr.tmux.RUnlock()
	for _, host := range logr.targetHosts {
		e, s := host.IsLevelEnabled(lvl)
		if e {
			status.Enabled = true
			if s {
				status.Stacktrace = true
				break // if both enabled then no sense checking more targets
			}
		}
	}

	// Cache and return the result.
	if err := logr.lvlCache.put(lvl.ID, status); err != nil {
		logr.ReportError(err)
		return LevelStatus{}
	}
	return status
}

// HasTargets returns true only if at least one target exists within the Logr.
func (logr *Logr) HasTargets() bool {
	logr.tmux.RLock()
	defer logr.tmux.RUnlock()
	return len(logr.targetHosts) > 0
}

// TargetInfo provides name and type for a Target.
type TargetInfo struct {
	Name string
	Type string
}

// TargetInfos enumerates all the targets added to this Logr.
// The resulting slice represents a snapshot at time of calling.
func (logr *Logr) TargetInfos() []TargetInfo {
	infos := make([]TargetInfo, 0)

	logr.tmux.RLock()
	defer logr.tmux.RUnlock()

	for _, host := range logr.targetHosts {
		inf := TargetInfo{
			Name: host.String(),
			Type: fmt.Sprintf("%T", host.target),
		}
		infos = append(infos, inf)
	}
	return infos
}

// RemoveTargets safely removes one or more targets based on the filtering method.
// f should return true to delete the target, false to keep it.
// When removing a target, best effort is made to write any queued log records before
// closing, with cxt determining how much time can be spent in total.
// Note, keep the timeout short since this method blocks certain logging operations.
func (logr *Logr) RemoveTargets(cxt context.Context, f func(ti TargetInfo) bool) error {
	errs := merror.New()
	hosts := make([]*TargetHost, 0)

	logr.tmux.Lock()
	defer logr.tmux.Unlock()

	for _, host := range logr.targetHosts {
		inf := TargetInfo{
			Name: host.String(),
			Type: fmt.Sprintf("%T", host.target),
		}
		if f(inf) {
			if err := host.Shutdown(cxt); err != nil {
				errs.Append(err)
			}
		} else {
			hosts = append(hosts, host)
		}
	}

	logr.targetHosts = hosts
	logr.ResetLevelCache()

	return errs.ErrorOrNil()
}

// ResetLevelCache resets the cached results of `IsLevelEnabled`. This is
// called any time a Target is added or a target's level is changed.
func (logr *Logr) ResetLevelCache() {
	logr.lvlCache.clear()
}

// enqueue adds a log record to the logr queue. If the queue is full then
// this function either blocks or the log record is dropped, depending on
// the result of calling `OnQueueFull`.
func (logr *Logr) enqueue(rec *LogRec) {
	select {
	case logr.in <- rec:
	default:
		if logr.options.onQueueFull != nil && logr.options.onQueueFull(rec, cap(logr.in)) {
			return // drop the record
		}
		select {
		case <-time.After(logr.options.enqueueTimeout):
			logr.ReportError(fmt.Errorf("enqueue timed out for log rec [%v]", rec))
		case logr.in <- rec: // block until success or timeout
		}
	}
}

// exit is called by one of the FatalXXX style APIS. If `logr.OnExit` is not nil
// then that method is called, otherwise the default behavior is to shut down this
// Logr cleanly then call `os.Exit(code)`.
func (logr *Logr) exit(code int) {
	if logr.options.onExit != nil {
		logr.options.onExit(code)
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
	if logr.options.onPanic != nil {
		logr.options.onPanic(err)
		return
	}

	if err := logr.Shutdown(); err != nil {
		logr.ReportError(err)
	}
	panic(err)
}

// Flush blocks while flushing the logr queue and all target queues, by
// writing existing log records to valid targets.
// Any attempts to add new log records will block until flush is complete.
// `logr.FlushTimeout` determines how long flush can execute before
// timing out. Use `IsTimeoutError` to determine if the returned error is
// due to a timeout.
func (logr *Logr) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), logr.options.flushTimeout)
	defer cancel()
	return logr.FlushWithTimeout(ctx)
}

// Flush blocks while flushing the logr queue and all target queues, by
// writing existing log records to valid targets.
// Any attempts to add new log records will block until flush is complete.
// Use `IsTimeoutError` to determine if the returned error is
// due to a timeout.
func (logr *Logr) FlushWithTimeout(ctx context.Context) error {
	if !logr.HasTargets() {
		return nil
	}

	if logr.IsShutdown() {
		return errors.New("Flush called on shut down Logr")
	}

	rec := newFlushLogRec(logr.NewLogger())
	logr.enqueue(rec)

	select {
	case <-ctx.Done():
		return newTimeoutError("logr queue flush timeout")
	case <-rec.flush:
	}
	return nil
}

// IsShutdown returns true if this Logr instance has been shut down.
// No further log records can be enqueued and no targets added after
// shutdown.
func (logr *Logr) IsShutdown() bool {
	return atomic.LoadInt32(&logr.shutdown) != 0
}

// Shutdown cleanly stops the logging engine after making best efforts
// to flush all targets. Call this function right before application
// exit - logr cannot be restarted once shut down.
// `logr.ShutdownTimeout` determines how long shutdown can execute before
// timing out. Use `IsTimeoutError` to determine if the returned error is
// due to a timeout.
func (logr *Logr) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), logr.options.shutdownTimeout)
	defer cancel()
	return logr.ShutdownWithTimeout(ctx)
}

// Shutdown cleanly stops the logging engine after making best efforts
// to flush all targets. Call this function right before application
// exit - logr cannot be restarted once shut down.
// Use `IsTimeoutError` to determine if the returned error is due to a
// timeout.
func (logr *Logr) ShutdownWithTimeout(ctx context.Context) error {
	if atomic.SwapInt32(&logr.shutdown, 1) != 0 {
		return errors.New("Shutdown called again after shut down")
	}

	logr.ResetLevelCache()
	logr.stopMetricsUpdater()

	close(logr.quit)

	errs := merror.New()

	// Wait for read loop to exit
	select {
	case <-ctx.Done():
		errs.Append(newTimeoutError("logr queue shutdown timeout"))
	case <-logr.done:
	}

	// logr.in channel should now be drained to targets and no more log records
	// can be added.
	logr.tmux.RLock()
	defer logr.tmux.RUnlock()
	for _, host := range logr.targetHosts {
		err := host.Shutdown(ctx)
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
	logr.incErrorCounter()

	if logr.options.onLoggerError == nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	logr.options.onLoggerError(fmt.Errorf("%v", err))
}

// BorrowBuffer borrows a buffer from the pool. Release the buffer to reduce garbage collection.
func (logr *Logr) BorrowBuffer() *bytes.Buffer {
	if logr.options.disableBufferPool {
		return &bytes.Buffer{}
	}
	return logr.bufferPool.Get().(*bytes.Buffer)
}

// ReleaseBuffer returns a buffer to the pool to reduce garbage collection. The buffer is only
// retained if less than MaxPooledBuffer.
func (logr *Logr) ReleaseBuffer(buf *bytes.Buffer) {
	if !logr.options.disableBufferPool && buf.Cap() < logr.options.maxPooledBuffer {
		buf.Reset()
		logr.bufferPool.Put(buf)
	}
}

// start selects on incoming log records until shutdown record is received.
// Incoming log records are fanned out to all log targets.
func (logr *Logr) start() {
	defer func() {
		if r := recover(); r != nil {
			logr.ReportError(r)
			go logr.start()
		} else {
			close(logr.done)
		}
	}()

	for {
		var rec *LogRec
		select {
		case rec = <-logr.in:
			if rec.flush != nil {
				logr.flush(rec.flush)
			} else {
				rec.prep()
				logr.fanout(rec)
			}
		case <-logr.quit:
			return
		}
	}
}

// fanout pushes a LogRec to all targets.
func (logr *Logr) fanout(rec *LogRec) {
	var host *TargetHost
	defer func() {
		if r := recover(); r != nil {
			logr.ReportError(fmt.Errorf("fanout failed for target %s, %v", host.String(), r))
		}
	}()

	var logged bool

	logr.tmux.RLock()
	defer logr.tmux.RUnlock()
	for _, host = range logr.targetHosts {
		if enabled, _ := host.IsLevelEnabled(rec.Level()); enabled {
			host.Log(rec)
			logged = true
		}
	}

	if logged {
		logr.incLoggedCounter()
	}
}

// flush drains the queue and notifies when done.
func (logr *Logr) flush(done chan<- struct{}) {
	// first drain the logr queue.
loop:
	for {
		var rec *LogRec
		select {
		case rec = <-logr.in:
			if rec.flush == nil {
				rec.prep()
				logr.fanout(rec)
			}
		default:
			break loop
		}
	}

	logger := logr.NewLogger()

	// drain all the targets; block until finished.
	logr.tmux.RLock()
	defer logr.tmux.RUnlock()
	for _, host := range logr.targetHosts {
		rec := newFlushLogRec(logger)
		host.Log(rec)
		<-rec.flush
	}
	done <- struct{}{}
}
