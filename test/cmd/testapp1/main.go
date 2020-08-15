package main

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"
	"sync/atomic"
	"time"

	"github.com/mattermost/logr"
	"github.com/mattermost/logr/format"
	"github.com/mattermost/logr/target"
	"github.com/mattermost/logr/test"
)

const (
	// GOROUTINES is the number of goroutines
	GOROUTINES = 10
	// LOOPS is the number of loops per goroutine.
	LOOPS = 10000
)

var lgr = &logr.Logr{
	MaxQueueSize:      1000,
	OnLoggerError:     handleLoggerError,
	OnQueueFull:       handleQueueFull,
	OnTargetQueueFull: handleTargetQueueFull,
}

var (
	errorCount           uint32
	queueFullCount       uint32
	targetQueueFullCount uint32
)

func handleLoggerError(err error) {
	atomic.AddUint32(&errorCount, 1)
	fmt.Fprintln(os.Stderr, "!!!!! OnLoggerError -- ", err)
}

func handleQueueFull(rec *logr.LogRec, maxQueueSize int) bool {
	fmt.Fprintf(os.Stderr, "!!!!! OnQueueFull - Max size %d. Count %d. Blocking...\n",
		maxQueueSize, atomic.AddUint32(&queueFullCount, 1))
	return false
}

func handleTargetQueueFull(target logr.Target, rec *logr.LogRec, maxQueueSize int) bool {
	fmt.Fprintf(os.Stderr, "!!!!! OnTargetQueueFull - (%v). Max size %d. Count %d. Blocking...\n",
		target, maxQueueSize, atomic.AddUint32(&targetQueueFullCount, 1))
	return false
}

func main() {
	// add metrics
	collector := test.NewTestMetricsCollector()
	if err := lgr.SetMetricsCollector(collector); err != nil {
		panic(err)
	}
	lgr.MetricsUpdateFreqMillis = 1000

	// create writer target to stdout
	var t logr.Target
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.JSON{}
	t = target.NewWriterTarget(filter, formatter, os.Stdout, 1000)
	t.SetName("stdout")
	_ = lgr.AddTarget(t)

	// create writer target to /dev/null
	t = target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
	t.SetName("discard")
	_ = lgr.AddTarget(t)

	// create syslog target to local using custom filter.
	lvl := logr.Level{ID: 77, Name: "Summary", Stacktrace: false}
	fltr := &logr.CustomFilter{}
	fltr.Add(lvl)
	params := &target.SyslogParams{Priority: syslog.LOG_WARNING | syslog.LOG_DAEMON, Tag: "logrtestapp"}
	t, err := target.NewSyslogTarget(fltr, formatter, params, 1000)
	t.SetName("syslog")
	if err != nil {
		panic(err)
	}
	_ = lgr.AddTarget(t)

	done := make(chan struct{})
	targetNames := []string{"_logr", "stdout", "discard", "syslog"}
	go startMetricsUpdater(targetNames, collector, done)

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: GOROUTINES,
		Loops:      LOOPS,
		GoodToken:  "Woot!",
		BadToken:   "XXX!!XXX",
		Lvl:        logr.Error,
		Delay:      time.Millisecond * 1,
	}
	logged, filtered := test.DoSomeLogging(cfg)

	err = lgr.Flush()
	if err != nil {
		panic(err)
	}

	logged2, filtered2 := test.DoSomeLogging(cfg)

	lgr.NewLogger().Logf(lvl, "Logr test completed. errors=%d, queueFull=%d, targetFull=%d",
		atomic.LoadUint32(&errorCount),
		atomic.LoadUint32(&queueFullCount),
		atomic.LoadUint32(&targetQueueFullCount))

	close(done)
	err = lgr.Shutdown()
	if err != nil {
		panic(err)
	}

	for _, name := range targetNames {
		printMetrics(name, collector)
	}

	fmt.Fprintf(os.Stderr, "Exiting normally. logged=%d, filtered=%d, errors=%d, queueFull=%d, targetFull=%d\n",
		logged+logged2,
		filtered+filtered2,
		atomic.LoadUint32(&errorCount),
		atomic.LoadUint32(&queueFullCount),
		atomic.LoadUint32(&targetQueueFullCount))
}

func startMetricsUpdater(targets []string, collector *test.TestMetricsCollector, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		case <-time.After(5 * time.Second):
			for _, name := range targets {
				printMetrics(name, collector)
			}
		}
	}
}

func printMetrics(target string, collector *test.TestMetricsCollector) {
	metrics := collector.Get(target)

	fmt.Fprintf(os.Stderr, "\n%s metrics:\n\tqueue: %g\n\tlogged: %g\n\terrors: %g\n\tdropped: %g\n\tblocked: %g\n",
		target, metrics.QueueSize, metrics.Logged, metrics.Errors, metrics.Dropped, metrics.Blocked)
}
