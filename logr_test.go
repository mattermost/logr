package logr_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/test"
)

func TestFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &format.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(filter, formatter, buf, 3000)
	target.Delay = time.Millisecond * 2
	lgr := &logr.Logr{}
	err := lgr.AddTarget(target)
	if err != nil {
		t.Error(err)
	}

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 20,
		Loops:      100,
		Lvl:        logr.Error,
	}
	test.DoSomeLogging(cfg)
	logger := lgr.NewLogger()
	logger.Info("Last entry @!!@")

	start := time.Now()

	// blocks until flush is finished.
	err = lgr.Flush()
	if err != nil {
		t.Error(err)
	}

	dur := time.Since(start)
	t.Logf("Flush duration: %v", dur)

	output := buf.String()
	if !strings.Contains(output, "@!!@") {
		t.Errorf("missing last log record")
	}

	// make sure logging can continue after flush.
	test.DoSomeLogging(cfg)
	logger.Info("Last entry %^^%")

	// blocks until flush is finished.
	err = lgr.Flush()
	if err != nil {
		t.Error(err)
	}

	output = buf.String()
	if !strings.Contains(output, "%^^%") {
		t.Errorf("missing last log record")
	}
}
