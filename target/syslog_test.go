// +build !windows,!nacl,!plan9

package target_test

import (
	"fmt"
	"log/syslog"
	"os"
	"testing"
	"time"

	"github.com/mattermost/logr"
	"github.com/mattermost/logr/format"
	"github.com/mattermost/logr/target"
	"github.com/mattermost/logr/test"
	"github.com/stretchr/testify/require"
)

func ExampleSyslog() {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	params := &target.SyslogParams{Network: "", Raddr: "", Priority: syslog.LOG_WARNING | syslog.LOG_DAEMON, Tag: "logrtest"}
	t, err := target.NewSyslogTarget(filter, formatter, params, 1000)
	if err != nil {
		panic(err)
	}
	_ = lgr.AddTarget(t)

	logger := lgr.NewLogger().WithField("name", "wiggin")

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
	plain := &format.Plain{Delim: " | ", DisableTimestamp: true}
	syslogger(t, plain)
}

func syslogger(t *testing.T, formatter logr.Formatter) {
	opt := logr.OnLoggerError(func(err error) {
		t.Error(err)
	})
	lgr, err := logr.New(opt)
	require.NoError(t, err)

	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Panic}
	params := &target.SyslogParams{Network: "", Raddr: "", Priority: syslog.LOG_WARNING | syslog.LOG_DAEMON, Tag: "logrtest"}
	target, err := target.NewSyslogTarget(filter, formatter, params, 1000)
	if err != nil {
		t.Error(err)
	}
	_ = lgr.AddTarget(target)

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
	if err != nil {
		t.Error(err)
	}
}
