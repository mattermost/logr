package logr_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(buf, 2)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "flushTest", filter, formatter, 3000)
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

	output = buf.String()
	if !strings.Contains(output, "%^^%") {
		t.Errorf("missing last log record")
	}
}

func TestFlushAfterShutdown(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(buf, 2)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "flushTest", filter, formatter, 3000)
	require.NoError(t, err)

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 20,
		Loops:      100,
		Lvl:        logr.Error,
	}
	test.DoSomeLogging(cfg)

	err = lgr.Shutdown()
	require.NoError(t, err)

	// Should error since shutdown already called. Shouldn't crash.
	err = lgr.Flush()
	require.Error(t, err)
}

func TestLogAfterShutdown(t *testing.T) {
	buf := &bytes.Buffer{}
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	target := test.NewSlowTarget(buf, 2)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "shutdownTest", filter, formatter, 3000)
	require.NoError(t, err)

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 20,
		Loops:      100,
		Lvl:        logr.Error,
	}
	test.DoSomeLogging(cfg)

	err = lgr.Shutdown()
	require.NoError(t, err)

	// Should NOP since shutdown already called. Shouldn't crash.
	logger := lgr.NewLogger().With(logr.String("test", "yes"))
	logger.Info("This shouldn't get logged")

	// Second shutdown should error, but not crash.
	err = lgr.Shutdown()
	require.Error(t, err)

	output := buf.String()
	if strings.Contains(output, "This shouldn't get logged") {
		t.Errorf("log record should not appear after shutdown")
	}
}

func TestRemoveTarget(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	lgr, _ := logr.New()

	buf1 := &bytes.Buffer{}
	target1 := test.NewSlowTarget(buf1, 2)
	err := lgr.AddTarget(target1, "t1", filter, formatter, 3000)
	require.NoError(t, err)

	buf2 := &bytes.Buffer{}
	target2 := test.NewSlowTarget(buf2, 2)
	err = lgr.AddTarget(target2, "t2", filter, formatter, 3000)
	require.NoError(t, err)

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
