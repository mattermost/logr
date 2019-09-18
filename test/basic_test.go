package logr_test

import (
	"bytes"
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
	buf := &buffer{}
	target := &target.Writer{Level: logr.WarnLevel, Fmtr: &format.Plain{Delim: " | "}, Out: buf, MaxQueued: 1000}
	logr.AddTarget(target)

	logger := logr.NewLogger().WithField("", "")

	logger.Errorf("the erroneous data is %s", stringRnd(10))
	logger.Warnf("strange data: %s", stringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	output := buf.String()
	fmt.Println(output)
}

func TestBasic(t *testing.T) {
	buf := &buffer{}
	target := &target.Writer{Level: logr.WarnLevel, Fmtr: &format.Plain{Delim: " | "}, Out: buf, MaxQueued: 1000}
	logr.AddTarget(target)

	wg := sync.WaitGroup{}
	var id int32 = 0

	runner := func(loops int) {
		defer wg.Done()
		tid := atomic.AddInt32(&id, 1)
		logger := logr.NewLogger().WithFields(logr.Fields{"id": tid, "rnd": rand.Intn(100)})

		for i := 0; i < loops; i++ {
			logger.Debug("XXX")
			logger.Trace("XXX")
			logger.Errorf("count:%d -- the erroneous data is %s", i, stringRnd(10))
			logger.Warnf("strange data: %s", stringRnd(5))
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

//
// Pseudo-random string generator
//
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func stringRnd(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

//
// Simple buffer implementing io.Writer
//
type buffer struct {
	mux sync.Mutex
	buf bytes.Buffer
}

func (b *buffer) Write(data []byte) (int, error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.Write(data)
}

func (b *buffer) String() string {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.String()
}
