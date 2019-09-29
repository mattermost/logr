package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

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

func main() {
	lgr := &logr.Logr{MaxQueueSize: 1000}
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	t := target.NewWriterTarget(filter, formatter, os.Stdout, 1000)
	lgr.AddTarget(t)

	t = target.NewWriterTarget(filter, formatter, ioutil.Discard, 1000)
	lgr.AddTarget(t)

	wg := sync.WaitGroup{}
	var id int32
	var filterCount int32
	var logCount int32

	runner := func(loops int) {
		defer wg.Done()
		tid := atomic.AddInt32(&id, 1)
		logger := lgr.NewLogger().WithFields(logr.Fields{"id": tid, "rnd": rand.Intn(100)})

		for i := 1; i <= loops; i++ {
			atomic.AddInt32(&filterCount, 2)
			logger.Debug("XXX")
			logger.Trace("XXX")

			lc := atomic.AddInt32(&logCount, 1)
			logger.Errorf("count:%d -- random data: %s", lc, test.StringRnd(10))

		}
	}

	start := time.Now()

	for i := 0; i < GOROUTINES; i++ {
		wg.Add(1)
		go runner(LOOPS)
	}
	wg.Wait()

	end := time.Now()

	err := lgr.Shutdown()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(atomic.LoadInt32(&logCount), " log entries output.")
	fmt.Println(atomic.LoadInt32(&filterCount), " log entries filtered.")
	fmt.Println(end.Sub(start).String())
}
