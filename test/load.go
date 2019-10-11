package test

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wiggin77/logr"
)

// DoSomeLoggingCfg is configuration for `DoSomeLogging` utility.
type DoSomeLoggingCfg struct {
	// Lgr is a preconfigured Logr instance.
	Lgr *logr.Logr
	// Goroutines is number of goroutines to start.
	Goroutines int
	// Loops is number of loops per goroutine.
	Loops int
	// GoodToken is some text that is output for log statements that
	// should be output.
	GoodToken string
	// BadToken is text that is output for log statements that should be
	// filtered out.
	BadToken string
	// Lvl is the Level to use for log statements.
	Lvl logr.Level
	// Delay is amount of time to pause between loops.
	Delay time.Duration
}

// DoSomeLogging performs some concurrent logging on a preconfigured Logr.
func DoSomeLogging(cfg DoSomeLoggingCfg) (logged int32, filtered int32) {
	wg := sync.WaitGroup{}
	var id int32
	var filterCount int32
	var logCount int32

	runner := func(loops int) {
		defer wg.Done()
		tid := atomic.AddInt32(&id, 1)
		logger := cfg.Lgr.NewLogger().WithFields(logr.Fields{"id": tid, "rnd": rand.Intn(100)})

		for i := 1; i <= loops; i++ {
			if cfg.Lvl.ID() < logr.Trace.ID() {
				atomic.AddInt32(&filterCount, 1)
				logger.Log(logr.Trace, "This should not be output. ", cfg.BadToken)
			}
			lc := atomic.AddInt32(&logCount, 1)
			logger.Logf(cfg.Lvl, "count:%d -- %s -- This is some sample text.", lc, cfg.GoodToken)

			if cfg.Delay > 0 {
				time.Sleep(cfg.Delay)
			}
		}
	}

	for i := 0; i < cfg.Goroutines; i++ {
		wg.Add(1)
		go runner(cfg.Loops)
	}
	wg.Wait()

	return atomic.LoadInt32(&logCount), atomic.LoadInt32(&filterCount)
}
