package test

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wiggin77/logr"
)

// DoSomeLogging performs some concurrent logging on a preconfigured Logr.
func DoSomeLogging(lgr *logr.Logr, goroutines int, loops int, goodToken string, badToken string) {
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
			logger.Debug("This is some debug log output. ", badToken)
			logger.Trace("A trace line for logging. ", badToken)

			lc := atomic.AddInt32(&logCount, 1)
			logger.Warnf("count:%d -- %s -- random data: %s", lc, goodToken, StringRnd(10))
			time.Sleep(1 * time.Millisecond)
		}
	}

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go runner(loops)
	}
	wg.Wait()

	end := time.Now()
	lgr.NewLogger().Infof("test ending at %v", end)

	err := lgr.Shutdown()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(atomic.LoadInt32(&logCount), " log entries output.")
	fmt.Println(atomic.LoadInt32(&filterCount), " log entries filtered.")
	fmt.Println(end.Sub(start).String())
}

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// StringRnd returns a pseudo-random string of the specified length.
func StringRnd(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Buffer is a simple buffer implementing io.Writer
type Buffer struct {
	mux sync.Mutex
	buf bytes.Buffer
}

// Write adds data to the buffer.
func (b *Buffer) Write(data []byte) (int, error) {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.Write(data)
}

// String returns the buffer as a string.
func (b *Buffer) String() string {
	b.mux.Lock()
	defer b.mux.Unlock()
	return b.buf.String()
}
