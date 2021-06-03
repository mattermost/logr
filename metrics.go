package logr

import "time"

const (
	DefMetricsUpdateFreqMillis = 15000 // 15 seconds
)

// Counter is a simple metrics sink that can only increment a value.
// Implementations are external to Logr and provided via `MetricsCollector`.
type Counter interface {
	// Inc increments the counter by 1. Use Add to increment it by arbitrary non-negative values.
	Inc()
	// Add adds the given value to the counter. It panics if the value is < 0.
	Add(float64)
}

// Gauge is a simple metrics sink that can receive values and increase or decrease.
// Implementations are external to Logr and provided via `MetricsCollector`.
type Gauge interface {
	// Set sets the Gauge to an arbitrary value.
	Set(float64)
	// Add adds the given value to the Gauge. (The value can be negative, resulting in a decrease of the Gauge.)
	Add(float64)
	// Sub subtracts the given value from the Gauge. (The value can be negative, resulting in an increase of the Gauge.)
	Sub(float64)
}

// MetricsCollector provides a way for users of this Logr package to have metrics pushed
// in an efficient way to any backend, e.g. Prometheus.
// For each target added to Logr, the supplied MetricsCollector will provide a Gauge
// and Counters that will be called frequently as logging occurs.
type MetricsCollector interface {
	// QueueSizeGauge returns a Gauge that will be updated by the named target.
	QueueSizeGauge(target string) (Gauge, error)
	// LoggedCounter returns a Counter that will be incremented by the named target.
	LoggedCounter(target string) (Counter, error)
	// ErrorCounter returns a Counter that will be incremented by the named target.
	ErrorCounter(target string) (Counter, error)
	// DroppedCounter returns a Counter that will be incremented by the named target.
	DroppedCounter(target string) (Counter, error)
	// BlockedCounter returns a Counter that will be incremented by the named target.
	BlockedCounter(target string) (Counter, error)
}

// TargetWithMetrics is a target that provides metrics.
type TargetWithMetrics interface {
	EnableMetrics(collector MetricsCollector, updateFreqMillis int64) error
}

type metrics struct {
	collector      MetricsCollector
	queueSizeGauge Gauge
	loggedCounter  Counter
	errorCounter   Counter
	done           chan struct{}
}

// initMetrics initializes metrics collection.
func (lgr *Logr) initMetrics() {
	if lgr.options.metricsCollector == nil {
		return
	}

	metrics := &metrics{
		collector: lgr.options.metricsCollector,
		done:      make(chan struct{}),
	}
	metrics.queueSizeGauge, _ = lgr.options.metricsCollector.QueueSizeGauge("_logr")
	metrics.loggedCounter, _ = lgr.options.metricsCollector.LoggedCounter("_logr")
	metrics.errorCounter, _ = lgr.options.metricsCollector.ErrorCounter("_logr")

	lgr.metrics = metrics

	go lgr.startMetricsUpdater()
}

func (lgr *Logr) setQueueSizeGauge(val float64) {
	if lgr.metrics != nil {
		lgr.metrics.queueSizeGauge.Set(val)
	}
}

func (lgr *Logr) incLoggedCounter() {
	if lgr.metrics != nil {
		lgr.metrics.loggedCounter.Inc()
	}
}

func (lgr *Logr) incErrorCounter() {
	if lgr.metrics != nil {
		lgr.metrics.errorCounter.Inc()
	}
}

// startMetricsUpdater updates the metrics for any polled values every `metricsUpdateFreqSecs` seconds until
// logr is closed.
func (lgr *Logr) startMetricsUpdater() {
	for {
		select {
		case <-lgr.metrics.done:
			return
		case <-time.After(time.Duration(lgr.options.metricsUpdateFreqMillis) * time.Millisecond):
			lgr.setQueueSizeGauge(float64(len(lgr.in)))
		}
	}
}

func (lgr *Logr) stopMetricsUpdater() {
	if lgr.metrics != nil {
		close(lgr.metrics.done)
	}
}
