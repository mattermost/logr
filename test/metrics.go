package test

import (
	"sync"

	"github.com/mattermost/logr"
)

type TestMetrics struct {
	QueueSize float64
	Logged    float64
	Errors    float64
	Dropped   float64
	Blocked   float64
}

type TestMetricsCollector struct {
	queueSizeGauges map[string]*TestGauge
	loggedCounters  map[string]*TestCounter
	errorCounters   map[string]*TestCounter
	droppedCounters map[string]*TestCounter
	blockedCounters map[string]*TestCounter
}

func NewTestMetricsCollector() *TestMetricsCollector {
	return &TestMetricsCollector{
		queueSizeGauges: make(map[string]*TestGauge),
		loggedCounters:  make(map[string]*TestCounter),
		errorCounters:   make(map[string]*TestCounter),
		droppedCounters: make(map[string]*TestCounter),
		blockedCounters: make(map[string]*TestCounter),
	}
}

func (c *TestMetricsCollector) Get(target string) TestMetrics {
	return TestMetrics{
		QueueSize: c.queueSizeGauges[target].get(),
		Logged:    c.loggedCounters[target].get(),
		Errors:    c.errorCounters[target].get(),
		Dropped:   c.droppedCounters[target].get(),
		Blocked:   c.blockedCounters[target].get(),
	}
}

func (c *TestMetricsCollector) QueueSizeGauge(target string) (logr.Gauge, error) {
	gauge, ok := c.queueSizeGauges[target]
	if !ok {
		gauge = &TestGauge{}
		c.queueSizeGauges[target] = gauge
	}
	return gauge, nil
}

func (c *TestMetricsCollector) LoggedCounter(target string) (logr.Counter, error) {
	counter, ok := c.loggedCounters[target]
	if !ok {
		counter = &TestCounter{}
		c.loggedCounters[target] = counter
	}
	return counter, nil
}

func (c *TestMetricsCollector) ErrorCounter(target string) (logr.Counter, error) {
	counter, ok := c.errorCounters[target]
	if !ok {
		counter = &TestCounter{}
		c.errorCounters[target] = counter
	}
	return counter, nil
}

func (c *TestMetricsCollector) DroppedCounter(target string) (logr.Counter, error) {
	counter, ok := c.droppedCounters[target]
	if !ok {
		counter = &TestCounter{}
		c.droppedCounters[target] = counter
	}
	return counter, nil
}

func (c *TestMetricsCollector) BlockedCounter(target string) (logr.Counter, error) {
	counter, ok := c.blockedCounters[target]
	if !ok {
		counter = &TestCounter{}
		c.blockedCounters[target] = counter
	}
	return counter, nil
}

type TestGauge struct {
	val float64
	mux sync.Mutex
}

func (g *TestGauge) get() float64 {
	if g == nil {
		return 0
	}

	g.mux.Lock()
	defer g.mux.Unlock()
	return g.val
}

func (g *TestGauge) Set(val float64) {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.val = val
}

func (g *TestGauge) Add(val float64) {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.val += val
}

func (g *TestGauge) Sub(val float64) {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.val -= val
}

type TestCounter struct {
	val float64
	mux sync.Mutex
}

func (c *TestCounter) get() float64 {
	if c == nil {
		return 0
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	return c.val
}

func (c *TestCounter) Inc() {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.val++
}

func (c *TestCounter) Add(val float64) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.val += val
}
