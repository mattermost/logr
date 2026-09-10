package logr_test

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/require"
)

// countingTarget records whether it was initialized and shut down, so a test
// can tell what ReplaceTargets did with a target it created.
type countingTarget struct {
	initErr      error
	inits        int
	shutdowns    int
	writtenBytes int
}

func (t *countingTarget) Init() error {
	t.inits++
	return t.initErr
}

func (t *countingTarget) Write(p []byte, rec *logr.LogRec) (int, error) {
	t.writtenBytes += len(p)
	return len(p), nil
}

func (t *countingTarget) Shutdown() error {
	t.shutdowns++
	return nil
}

func spec(target logr.Target, name string) logr.TargetSpec {
	return logr.TargetSpec{
		Target:    target,
		Name:      name,
		Filter:    &logr.StdFilter{Lvl: logr.Info},
		Formatter: &formatters.Plain{},
	}
}

func newReplaceLogr(t *testing.T) *logr.Logr {
	t.Helper()

	lgr, err := logr.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = lgr.Shutdown()
	})
	return lgr
}

func TestReplaceTargets(t *testing.T) {
	t.Run("adds the supplied targets", func(t *testing.T) {
		lgr := newReplaceLogr(t)

		first := &countingTarget{}
		second := &countingTarget{}
		require.NoError(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{
			spec(first, "first"), spec(second, "second"),
		}))

		require.Len(t, lgr.TargetInfos(), 2)
		require.Equal(t, 1, first.inits)
		require.Equal(t, 1, second.inits)
	})

	t.Run("shuts down the targets it replaces", func(t *testing.T) {
		lgr := newReplaceLogr(t)

		old := &countingTarget{}
		require.NoError(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(old, "old")}))

		replacement := &countingTarget{}
		require.NoError(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(replacement, "new")}))

		require.Equal(t, 1, old.shutdowns, "the replaced target should be shut down")
		require.Len(t, lgr.TargetInfos(), 1)
		require.Equal(t, "new", lgr.TargetInfos()[0].Name)
	})

	t.Run("an empty set removes every target", func(t *testing.T) {
		lgr := newReplaceLogr(t)

		old := &countingTarget{}
		require.NoError(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(old, "old")}))
		require.NoError(t, lgr.ReplaceTargets(context.Background(), nil))

		require.Empty(t, lgr.TargetInfos())
		require.Equal(t, 1, old.shutdowns)
	})
}

func TestReplaceTargetsLeavesExistingTargetsOnInitFailure(t *testing.T) {
	lgr := newReplaceLogr(t)

	keep := &countingTarget{}
	require.NoError(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(keep, "keep")}))

	good := &countingTarget{}
	broken := &countingTarget{initErr: errors.New("cannot reach daemon")}

	err := lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{
		spec(good, "good"), spec(broken, "broken"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "broken")
	require.Contains(t, err.Error(), "cannot reach daemon")

	require.Len(t, lgr.TargetInfos(), 1, "the working target should survive")
	require.Equal(t, "keep", lgr.TargetInfos()[0].Name)
	require.Zero(t, keep.shutdowns, "the working target should not be shut down")

	// The target created before the failure has to be shut down or its
	// goroutine and any connection it opened would leak.
	require.Equal(t, 1, good.shutdowns, "the already-created target should be shut down")
}

func TestReplaceTargetsKeepsLoggingAfterFailure(t *testing.T) {
	lgr := newReplaceLogr(t)

	keep := &countingTarget{}
	require.NoError(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(keep, "keep")}))

	broken := &countingTarget{initErr: errors.New("init failed")}
	require.Error(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(broken, "broken")}))

	logger := lgr.NewLogger()
	logger.Info("this record still has somewhere to go")
	require.NoError(t, lgr.Flush())

	require.NotZero(t, keep.writtenBytes, "records should still reach the surviving target")
}

func TestReplaceTargetsAfterShutdown(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	require.NoError(t, lgr.Shutdown())

	err = lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{
		spec(targets.NewWriterTarget(io.Discard), "late"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "shut down")
}

// countingGauge records how many times the metrics updater set a value, which
// is the observable side effect of that goroutine still running.
type countingGauge struct {
	mux  sync.Mutex
	sets int
}

func (g *countingGauge) Set(float64) {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.sets++
}

func (g *countingGauge) Add(float64) {}
func (g *countingGauge) Sub(float64) {}

func (g *countingGauge) count() int {
	g.mux.Lock()
	defer g.mux.Unlock()
	return g.sets
}

// gaugeCollector reuses the shared test collector and swaps in a gauge that
// counts writes. The Logr instance takes a queue size gauge of its own, so the
// gauges are kept per target name.
type gaugeCollector struct {
	*test.TestMetricsCollector

	mux    sync.Mutex
	gauges map[string]*countingGauge
}

func newGaugeCollector() *gaugeCollector {
	return &gaugeCollector{
		TestMetricsCollector: test.NewTestMetricsCollector(),
		gauges:               make(map[string]*countingGauge),
	}
}

func (c *gaugeCollector) gaugeFor(target string) *countingGauge {
	c.mux.Lock()
	defer c.mux.Unlock()

	gauge, ok := c.gauges[target]
	if !ok {
		gauge = &countingGauge{}
		c.gauges[target] = gauge
	}
	return gauge
}

func (c *gaugeCollector) QueueSizeGauge(target string) (logr.Gauge, error) {
	return c.gaugeFor(target), nil
}

// TestReplaceTargetsDoesNotLeakMetricsUpdaterOnInitFailure covers the cleanup
// of a target host that was only partly built. initMetrics starts the metrics
// updater before Target.Init is called, so a target that fails Init would
// otherwise leave that goroutine setting its gauge for the life of the process.
func TestReplaceTargetsDoesNotLeakMetricsUpdaterOnInitFailure(t *testing.T) {
	const updateFreqMillis = 250

	collector := newGaugeCollector()
	lgr, err := logr.New(logr.SetMetricsCollector(collector, updateFreqMillis))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = lgr.Shutdown()
	})

	broken := &countingTarget{initErr: errors.New("cannot reach daemon")}
	require.Error(t, lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(broken, "broken")}))

	gauge := collector.gaugeFor("broken")

	// Wait out several update periods, then confirm the gauge stops moving.
	time.Sleep(3 * updateFreqMillis * time.Millisecond)
	settled := gauge.count()

	time.Sleep(3 * updateFreqMillis * time.Millisecond)
	require.Equal(t, settled, gauge.count(),
		"the metrics updater for the target that failed Init is still running")
}

// TestReplaceTargetsDoesNotPublishAfterShutdown covers the window between
// building the replacement hosts and publishing them. ShutdownWithTimeout sets
// the shutdown flag well before it reads targetHosts, so a replacement set
// published after that read would keep running with nothing left to stop it.
func TestReplaceTargetsDoesNotPublishAfterShutdown(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)

	gate := make(chan struct{})
	blocking := &gatedTarget{ready: gate, entered: make(chan struct{})}

	done := make(chan error, 1)
	go func() {
		done <- lgr.ReplaceTargets(context.Background(), []logr.TargetSpec{spec(blocking, "late")})
	}()

	// Wait until the target is mid-Init, which is after ReplaceTargets has
	// checked IsShutdown but before it publishes.
	<-blocking.entered

	require.NoError(t, lgr.Shutdown())
	close(gate)

	require.NoError(t, <-done)
	require.Empty(t, lgr.TargetInfos(), "a target must not be published after shutdown")
	require.Equal(t, 1, blocking.shutdowns(), "the staged target should be shut down instead of published")
}

// gatedTarget blocks in Init until its gate is closed, so a test can interleave
// a shutdown with target creation.
type gatedTarget struct {
	ready   chan struct{}
	entered chan struct{}

	mux  sync.Mutex
	down int
}

func (t *gatedTarget) Init() error {
	close(t.entered)
	<-t.ready
	return nil
}

func (t *gatedTarget) Write(p []byte, rec *logr.LogRec) (int, error) { return len(p), nil }

func (t *gatedTarget) Shutdown() error {
	t.mux.Lock()
	defer t.mux.Unlock()
	t.down++
	return nil
}

func (t *gatedTarget) shutdowns() int {
	t.mux.Lock()
	defer t.mux.Unlock()
	return t.down
}
