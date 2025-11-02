//go:build !windows && !nacl && !plan9
// +build !windows,!nacl,!plan9

package targets_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/require"
)

func ExampleSyslog() {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	params := &targets.SyslogOptions{
		Host: "localhost",
		Port: 514,
		Tag:  "logrtest",
	}
	t, err := targets.NewSyslogTarget(params)
	if err != nil {
		panic(err)
	}
	err = lgr.AddTarget(t, "syslogTest", filter, formatter, 1000)
	if err != nil {
		panic(err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "wiggin")).Sugar()

	logger.Errorf("the erroneous data is %s", test.StringRnd(10))
	logger.Warnf("strange data: %s", test.StringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	err = lgr.Shutdown()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func TestSyslogPlain(t *testing.T) {
	plain := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	syslogger(t, plain)
}

func syslogger(t *testing.T, formatter logr.Formatter) {
	opt := logr.OnLoggerError(func(err error) {
		t.Error(err)
	})
	lgr, err := logr.New(opt)
	require.NoError(t, err)

	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Panic}
	params := &targets.SyslogOptions{
		Tag: "logrtest",
	}
	target, err := targets.NewSyslogTarget(params)
	require.NoError(t, err)

	err = lgr.AddTarget(target, "syslogTest2", filter, formatter, 1000)
	require.NoError(t, err)

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 3,
		Loops:      5,
		GoodToken:  "Woot!",
		BadToken:   "XXX!!XXX",
		Lvl:        logr.Warn,
		Delay:      time.Millisecond * 1,
	}
	test.DoSomeLogging(cfg)

	err = lgr.Shutdown()
	require.NoError(t, err)
}
