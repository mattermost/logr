package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"sync/atomic"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
)

// Settings
const (
	LOOPS  = 10000
	REPEAT = 10000
	QSIZE  = 10010
)

var opts = []logr.Option{
	logr.MaxQueueSize(QSIZE),
	logr.OnLoggerError(handleLoggerError),
	logr.OnQueueFull(handleQueueFull),
	logr.OnTargetQueueFull(handleTargetQueueFull),
}

var lgr, _ = logr.New(opts...)

var (
	errorCount           uint32
	queueFullCount       uint32
	targetQueueFullCount uint32
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

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
	// create writer target to stdout
	var t logr.Target
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	t = targets.NewWriterTarget(ioutil.Discard)
	err := lgr.AddTarget(t, "simple", filter, formatter, QSIZE)
	if err != nil {
		panic(err)
	}

	logger := lgr.NewLogger().WithFields(logr.Fields{"name": "Wiggin"})
	logger.Error("One test error.")

	var file *os.File

	flag.Parse()
	if *cpuprofile != "" {
		file, err = os.Create(*cpuprofile)
		if err != nil {
			panic(err)
		}
		_ = pprof.StartCPUProfile(file)
	}

	for r := 0; r < REPEAT; r++ {
		for i := 0; i < LOOPS; i++ {
			logger.Info("This is a message")
		}
		lgr.Flush()
	}

	fmt.Fprintf(os.Stdout, "Exiting normally. loops=%d, errors=%d, queueFull=%d, targetFull=%d\n",
		LOOPS*REPEAT,
		atomic.LoadUint32(&errorCount),
		atomic.LoadUint32(&queueFullCount),
		atomic.LoadUint32(&targetQueueFullCount))

	if file != nil {
		pprof.StopCPUProfile()
		file.Close()
	}
}
