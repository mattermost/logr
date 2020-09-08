package logr_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr"
	"github.com/mattermost/logr/format"
	"github.com/mattermost/logr/test"
	"github.com/stretchr/testify/assert"
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

func TestFlushAfterShutdown(t *testing.T) {
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

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}

	// Should error since shutdown already called. Shouldn't crash.
	err = lgr.Flush()
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestLogAfterShutdown(t *testing.T) {
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

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}

	// Should NOP since shutdown already called. Shouldn't crash.
	logger := lgr.NewLogger().WithField("test", "yes")
	logger.Info("This shouldn't get logged")

	// Second shutdown should error, but not crash.
	err = lgr.Shutdown()
	if err == nil {
		t.Errorf("Expected error calling shutdown after shutdown")
	}

	output := buf.String()
	if strings.Contains(output, "This shouldn't get logged") {
		t.Errorf("log record should not appear after shutdown")
	}
}

func TestRemoveTarget(t *testing.T) {
	formatter := &format.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}

	buf1 := &bytes.Buffer{}
	target1 := test.NewSlowTarget(filter, formatter, buf1, 3000)
	target1.SetName("t1")
	target1.Delay = time.Millisecond * 2

	buf2 := &bytes.Buffer{}
	target2 := test.NewSlowTarget(filter, formatter, buf2, 3000)
	target2.SetName("t2")
	target2.Delay = time.Millisecond * 2

	lgr := &logr.Logr{}
	err := lgr.AddTarget(target1, target2)
	assert.NoError(t, err)

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 20,
		Loops:      100,
		Lvl:        logr.Error,
	}
	test.DoSomeLogging(cfg)

	err = lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
		return ti.Name == "t2"
	})
	assert.NoError(t, err)

	tarr := lgr.TargetInfos()
	assert.Len(t, tarr, 1)
	assert.Equal(t, tarr[0].Name, "t1")

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}
}
