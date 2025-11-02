package logr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// Target represents a destination for log records such as file,
// database, TCP socket, etc.
type Target interface {
	// Init is called once to initialize the target.
	Init() error

	// Write outputs to this target's destination.
	Write(p []byte, rec *LogRec) (int, error)

	// Shutdown is called once to free/close any resources.
	// Target queue is already drained when this is called.
	Shutdown() error
}

type targetMetrics struct {
	queueSizeGauge Gauge
	loggedCounter  Counter
	errorCounter   Counter
	droppedCounter Counter
	blockedCounter Counter
}

type targetHostOptions struct {
	name         string
	filter       Filter
	formatter    Formatter
	maxQueueSize int
	metrics      *metrics
}

// TargetHost hosts and manages the lifecycle of a target.
// Incoming log records are queued and formatted before
// being passed to the target.
type TargetHost struct {
	target Target
	name   string

	filter    Filter
	formatter Formatter
	lvlCache  levelCache // per-target level cache for faster filtering

	in            chan *LogRec
	quit          chan context.Context // receives shutdown context to exit read loop
	done          chan struct{}        // closed when read loop exited
	targetMetrics *targetMetrics

	shutdown int32
}

func newTargetHost(target Target, options targetHostOptions) (*TargetHost, error) {
	host := &TargetHost{
		target:    target,
		name:      options.name,
		filter:    options.filter,
		formatter: options.formatter,
		lvlCache:  &syncMapLevelCache{}, // always use syncMap for per-target cache
		in:        make(chan *LogRec, options.maxQueueSize),
		quit:      make(chan context.Context, 1),
		done:      make(chan struct{}),
	}

	if host.name == "" {
		host.name = fmt.Sprintf("%T", target)
	}

	if host.filter == nil {
		host.filter = &StdFilter{Lvl: Fatal}
	}
	if host.formatter == nil {
		host.formatter = &DefaultFormatter{}
	}

	// Initialize the per-target level cache
	host.lvlCache.setup()

	err := host.initMetrics(options.metrics)
	if err != nil {
		return nil, err
	}

	err = target.Init()
	if err != nil {
		return nil, err
	}

	go host.start()

	return host, nil
}

func (h *TargetHost) initMetrics(metrics *metrics) error {
	if metrics == nil {
		return nil
	}

	var err error
	tmetrics := &targetMetrics{}

	if tmetrics.queueSizeGauge, err = metrics.collector.QueueSizeGauge(h.name); err != nil {
		return err
	}
	if tmetrics.loggedCounter, err = metrics.collector.LoggedCounter(h.name); err != nil {
		return err
	}
	if tmetrics.errorCounter, err = metrics.collector.ErrorCounter(h.name); err != nil {
		return err
	}
	if tmetrics.droppedCounter, err = metrics.collector.DroppedCounter(h.name); err != nil {
		return err
	}
	if tmetrics.blockedCounter, err = metrics.collector.BlockedCounter(h.name); err != nil {
		return err
	}
	h.targetMetrics = tmetrics

	updateFreqMillis := metrics.updateFreqMillis
	if updateFreqMillis == 0 {
		updateFreqMillis = DefMetricsUpdateFreqMillis
	}
	if updateFreqMillis < 250 {
		updateFreqMillis = 250 // don't peg the CPU
	}

	go h.startMetricsUpdater(updateFreqMillis)
	return nil
}

// IsLevelEnabled returns true if this target should emit logs for the specified level.
// Uses per-target cache for improved performance.
func (h *TargetHost) IsLevelEnabled(lvl Level) (enabled bool, level Level) {
	// Check cache first
	if status, ok := h.lvlCache.get(lvl.ID); ok {
		return status.Enabled, lvl
	}

	// Cache miss: check filter
	level, enabled = h.filter.GetEnabledLevel(lvl)

	// Cache the result (ignore error since it only fails for invalid level IDs)
	_ = h.lvlCache.put(lvl.ID, LevelStatus{Enabled: enabled})

	return enabled, level
}

// Shutdown stops processing log records after making best
// effort to flush queue.
func (h *TargetHost) Shutdown(ctx context.Context) error {
	if atomic.SwapInt32(&h.shutdown, 1) != 0 {
		return errors.New("targetHost shutdown called more than once")
	}

	// Send shutdown context to read loop instead of closing channel
	// This allows the read loop to receive the timeout context for drainQueue
	h.quit <- ctx

	// Wait for read loop to exit and queue to drain
	select {
	case <-ctx.Done():
		// Context timeout - proceed with shutdown
	case <-h.done:
		// Read loop exited and queue drained successfully
	}

	return h.target.Shutdown()
}

// Log queues a log record to be output to this target's destination.
func (h *TargetHost) Log(rec *LogRec) {
	// Check shutdown flag first
	if atomic.LoadInt32(&h.shutdown) != 0 {
		return
	}

	lgr := rec.Logger().Logr()

	// Attempt non-blocking send first to prevent hanging during shutdown,
	// with a fallback to blocking send with timeout if the queue is full.
	select {
	case h.in <- rec:
		// Successfully queued
		return
	default:
		// Queue full - check if we're shutting down
		if atomic.LoadInt32(&h.shutdown) != 0 {
			return // Don't block during shutdown
		}

		handler := lgr.options.onTargetQueueFull
		if handler != nil && handler(h.target, rec, cap(h.in)) {
			h.incDroppedCounter()
			return // drop the record
		}
		h.incBlockedCounter()

		// Try again with timeout
		select {
		case h.in <- rec:
			// Successfully queued after wait
		case <-time.After(lgr.options.enqueueTimeout):
			lgr.ReportError(fmt.Errorf("target enqueue timeout for log rec [%v]", rec))
		}
	}
}

func (h *TargetHost) setQueueSizeGauge(val float64) {
	if h.targetMetrics != nil {
		h.targetMetrics.queueSizeGauge.Set(val)
	}
}

func (h *TargetHost) incLoggedCounter() {
	if h.targetMetrics != nil {
		h.targetMetrics.loggedCounter.Inc()
	}
}

func (h *TargetHost) incErrorCounter() {
	if h.targetMetrics != nil {
		h.targetMetrics.errorCounter.Inc()
	}
}

func (h *TargetHost) incDroppedCounter() {
	if h.targetMetrics != nil {
		h.targetMetrics.droppedCounter.Inc()
	}
}

func (h *TargetHost) incBlockedCounter() {
	if h.targetMetrics != nil {
		h.targetMetrics.blockedCounter.Inc()
	}
}

// String returns a name for this target.
func (h *TargetHost) String() string {
	return h.name
}

// resetLevelCache clears the per-target level cache.
// Called when the Logr level cache is reset.
func (h *TargetHost) resetLevelCache() {
	h.lvlCache.clear()
}

// start accepts log records via In channel and writes to the
// supplied target, until Done channel signaled.
func (h *TargetHost) start() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "TargetHost.start -- ", r)
			go h.start()
		} else {
			close(h.done)
		}
	}()

	for {
		var rec *LogRec
		select {
		case rec = <-h.in:
			if rec.flush != nil {
				h.flush(rec.flush)
			} else {
				err := h.writeRec(rec)
				if err != nil {
					h.incErrorCounter()
					rec.Logger().Logr().ReportError(err)
				} else {
					h.incLoggedCounter()
				}
			}
		case shutdownCtx := <-h.quit:
			if shutdownCtx != nil {
				// close channel so that any goroutines created by the panic handler exit immediately
				close(h.quit)
				// Drain remaining records in queue with shutdown timeout if we have a context
				h.drainQueue(shutdownCtx)
			}
			return
		}
	}
}

func (h *TargetHost) writeRec(rec *LogRec) error {
	level, enabled := h.filter.GetEnabledLevel(rec.Level())
	if !enabled {
		// how did we get here?
		return fmt.Errorf("level %s not enabled for target %s", rec.Level().Name, h.name)
	}

	buf := rec.logger.lgr.BorrowBuffer()
	defer rec.logger.lgr.ReleaseBuffer(buf)

	buf, err := h.formatter.Format(rec, level, buf)
	if err != nil {
		return err
	}

	_, err = h.target.Write(buf.Bytes(), rec)
	return err
}

// startMetricsUpdater updates the metrics for any polled values every `updateFreqMillis` seconds until
// target is shut down.
func (h *TargetHost) startMetricsUpdater(updateFreqMillis int64) {
	for {
		select {
		case <-h.done:
			return
		case <-time.After(time.Duration(updateFreqMillis) * time.Millisecond):
			h.setQueueSizeGauge(float64(len(h.in)))
		}
	}
}

// drainQueue processes any remaining records in the queue during shutdown.
// It respects the provided context timeout to prevent indefinite blocking.
// Individual record processing errors are logged and the error counter is incremented.
func (h *TargetHost) drainQueue(ctx context.Context) {
	for {
		select {
		case rec := <-h.in:
			// Process remaining records
			if rec.flush != nil {
				h.flush(rec.flush)
			} else {
				err := h.writeRec(rec)
				if err != nil {
					h.incErrorCounter()
					rec.Logger().Logr().ReportError(err)
				} else {
					h.incLoggedCounter()
				}
			}
		case <-ctx.Done():
			// Shutdown timeout reached, abandon remaining records
			return
		default:
			// Queue is empty, safe to exit
			return
		}
	}
}

// flush drains the queue and notifies when done.
func (h *TargetHost) flush(done chan<- struct{}) {
	for {
		var rec *LogRec
		var err error
		select {
		case rec = <-h.in:
			// ignore any redundant flush records.
			if rec.flush == nil {
				err = h.writeRec(rec)
				if err != nil {
					h.incErrorCounter()
					rec.Logger().Logr().ReportError(err)
				}
			}
		default:
			done <- struct{}{}
			return
		}
	}
}
