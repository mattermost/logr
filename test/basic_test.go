package test

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/wiggin77/logr"

	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
)

func Example() {
	buf := &Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	target, err := target.NewWriterTarget(filter, formatter, buf, 1000)
	if err != nil {
		panic(err)
	}
	logr.AddTarget(target)

	logger := logr.NewLogger().WithField("", "")

	logger.Errorf("the erroneous data is %s", StringRnd(10))
	logger.Warnf("strange data: %s", StringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	output := buf.String()
	fmt.Println(output)
}

func TestBasic(t *testing.T) {
	buf := &Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	target, err := target.NewWriterTarget(filter, formatter, buf, 1000)
	if err != nil {
		t.Error(err)
	}
	logr.AddTarget(target)

	wg := sync.WaitGroup{}
	var id int32

	runner := func(loops int) {
		defer wg.Done()
		tid := atomic.AddInt32(&id, 1)
		logger := logr.NewLogger().WithFields(logr.Fields{"id": tid, "rnd": rand.Intn(100)})

		for i := 0; i < loops; i++ {
			logger.Debug("XXX")
			logger.Trace("XXX")
			logger.Errorf("count:%d -- the erroneous data is %s", i, StringRnd(10))
			logger.Warnf("strange data: %s", StringRnd(5))
			runtime.Gosched()
		}
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go runner(50)
	}
	wg.Wait()

	output := buf.String()
	fmt.Println(output)
	if strings.Contains(output, "XXX") {
		t.Errorf("wrong level(s) enabled")
	}
}
