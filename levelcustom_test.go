package logr_test

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/require"
)

var (
	LoginLevel  = logr.Level{ID: 250, Name: "login ", Stacktrace: false}
	LogoutLevel = logr.Level{ID: 251, Name: "logout", Stacktrace: false}
	BadLevel    = logr.Level{ID: logr.MaxLevelID + 1, Name: "invalid", Stacktrace: false}
)

func TestCustomLevel(t *testing.T) {
	lgr, _ := logr.New()
	buf := &test.Buffer{}

	// create a custom filter with custom levels.
	filter := &logr.CustomFilter{}
	filter.Add(LoginLevel, LogoutLevel)

	formatter := &formatters.Plain{Delim: " | "}
	tgr := targets.NewWriterTarget(buf)
	err := lgr.AddTarget(tgr, "customLevelTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger().With(
		logr.String("user", "Bob"),
		logr.String("role", "admin"),
	)

	logger.Log(LoginLevel, "LoginLevel will get logged")
	logger.Log(logr.Error, "XXX - won't be logged as Error was not added to custom filter.")
	logger.Debug("XXX - won't be logged")
	logger.Log(LogoutLevel, "LogoutLevel will get logged")

	err = lgr.Shutdown()
	require.NoError(t, err)

	output := buf.String()
	t.Log(output)

	if !strings.Contains(output, "LoginLevel") {
		t.Error("missing LoginLevel")
	}

	if !strings.Contains(output, "LogoutLevel") {
		t.Error("missing LogoutLevel")
	}

	if strings.Contains(output, "XXX") {
		t.Error("wrong level(s) output")
	}
}

func TestLevelIDTooLarge(t *testing.T) {
	var count int32
	opt := logr.OnLoggerError(func(err error) {
		atomic.AddInt32(&count, 1)
	})
	lgr, err := logr.New(opt)
	require.NoError(t, err)
	buf := &test.Buffer{}

	// create a custom filter with custom level.
	filter := &logr.CustomFilter{}
	filter.Add(BadLevel)

	formatter := &formatters.Plain{Delim: " | "}
	tgr := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(tgr, "levelTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger().With(logr.String("user", "Bob"), logr.String("role", "admin"))

	logger.Log(BadLevel, "this item will trigger OnLoggerError")

	err = lgr.Shutdown()
	require.NoError(t, err)

	if atomic.LoadInt32(&count) != 1 {
		t.Error("OnLoggerError should be called once")
	}
}
