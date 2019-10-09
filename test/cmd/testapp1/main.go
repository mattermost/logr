package main

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"os"
	"sync/atomic"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"
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
	queueFullCount       uint32
	targetQueueFullCount uint32
)

func handleLoggerError(err error) {
	panic(err)
}

func handleQueueFull(rec *logr.LogRec, maxQueueSize int) bool {
	fmt.Fprintf(os.Stderr, "!!!!! Logr queue full. Max size %d. Count %d. Blocking...\n",
		maxQueueSize, atomic.AddUint32(&queueFullCount, 1))
	return false
}

func handleTargetQueueFull(target logr.Target, rec *logr.LogRec, maxQueueSize int) bool {
	fmt.Fprintf(os.Stderr, "!!!!! Target queue full. Max size %d. Count %d. Blocking...\n",
		maxQueueSize, atomic.AddUint32(&targetQueueFullCount, 1))
	return false
}

func main() {
	// create writer target to stdout
	var t logr.Target
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	t = target.NewWriterTarget(filter, formatter, os.Stdout, 1000)
	lgr.AddTarget(t)

	// create writer target to /dev/null
	t = target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
	lgr.AddTarget(t)

	// create syslog target to local
	filter = &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Panic}
	params := &target.SyslogParams{Priority: syslog.LOG_WARNING | syslog.LOG_DAEMON, Tag: "logrtestapp"}
	t, err := target.NewSyslogTarget(filter, formatter, params, 1000)
	if err != nil {
		panic(err)
	}
	lgr.AddTarget(t)

	test.DoSomeLogging(lgr, GOROUTINES, LOOPS, "Good", "XXX!!XXX")
}
