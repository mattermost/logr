package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wiggin77/logr/level"

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
	t := &target.Writer{Level: level.Warn, Fmtr: &format.Plain{Delim: " | "}, Out: os.Stdout, MaxQueued: 1000}
	logr.AddTarget(t)

	t = &target.Writer{Level: level.Trace, Fmtr: &format.Plain{Delim: " | "}, Out: ioutil.Discard, MaxQueued: 1000}
	logr.AddTarget(t)

	wg := sync.WaitGroup{}
	var id int32
	var filterCount int32
	var logCount int32

	runner := func(loops int) {
		defer wg.Done()
		tid := atomic.AddInt32(&id, 1)
		logger := logr.NewLogger().WithFields(logr.Fields{"id": tid, "rnd": rand.Intn(100)})

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

	err := logr.Shutdown()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(atomic.LoadInt32(&logCount), " log entries output.")
	fmt.Println(atomic.LoadInt32(&filterCount), " log entries filtered.")
	fmt.Println(end.Sub(start).String())
}
