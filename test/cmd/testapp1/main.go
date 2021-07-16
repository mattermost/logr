package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync/atomic"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
)

const (
	// GOROUTINES is the number of goroutines
	GOROUTINES = 10
	// LOOPS is the number of loops per goroutine.
	LOOPS = 10000
	// QSIZE is the size of the Loge inbound queue.
	QSIZE = 1000
)

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
	collector := test.NewTestMetricsCollector()

	opts := []logr.Option{
		logr.MaxQueueSize(QSIZE),
		logr.OnLoggerError(handleLoggerError),
		logr.OnQueueFull(handleQueueFull),
		logr.OnTargetQueueFull(handleTargetQueueFull),
		logr.SetMetricsCollector(collector, 250),
	}

	lgr, err := logr.New(opts...)
	if err != nil {
		panic(err)
	}

	// create writer target to stdout
	var t logr.Target
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &formatters.JSON{EnableCaller: true}
	t = targets.NewWriterTarget(os.Stdout)
	err = lgr.AddTarget(t, "stdout", filter, formatter, 1000)
	if err != nil {
		panic(err)
	}

	// create writer target to /dev/null
	t = targets.NewWriterTarget(ioutil.Discard)
	err = lgr.AddTarget(t, "discard", filter, formatter, 1000)
	if err != nil {
		panic(err)
	}

	// create syslog target to local using custom filter.
	lvl := logr.Level{ID: 77, Name: "Summary", Stacktrace: false}
	fltr := &logr.CustomFilter{}
	fltr.Add(lvl)
	params := &targets.SyslogOptions{Tag: "logrtestapp"}
	t, err = targets.NewSyslogTarget(params)
	if err != nil {
		panic(err)
	}
	err = lgr.AddTarget(t, "syslog", fltr, formatter, 1000)
	if err != nil {
		panic(err)
	}

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

	lgr.NewLogger().Log(lvl, "Logr test completed.",
		logr.Uint32("errors", atomic.LoadUint32(&errorCount)),
		logr.Uint32("queueFull", atomic.LoadUint32(&queueFullCount)),
		logr.Uint32("targetFull", atomic.LoadUint32(&targetQueueFullCount)),
	)

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

	if atomic.LoadUint32(&errorCount) > 0 || atomic.LoadUint32(&queueFullCount) > 0 || atomic.LoadUint32(&targetQueueFullCount) > 0 {
		os.Exit(1)
	}
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
