package logr_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"
)

var (
	LoginLevel  = logr.Level{ID: 100, Name: "login ", Stacktrace: false}
	LogoutLevel = logr.Level{ID: 101, Name: "logout", Stacktrace: false}
)

func TestCustomLevel(t *testing.T) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}

	// create a custom filter with custom levels.
	filter := &logr.CustomFilter{}
	filter.Add(LoginLevel, LogoutLevel)

	formatter := &format.Plain{Delim: " | "}
	tgr := target.NewWriterTarget(filter, formatter, buf, 1000)
	lgr.AddTarget(tgr)

	logger := lgr.NewLogger().WithFields(logr.Fields{"user": "Bob", "role": "admin"})

	logger.Log(LoginLevel, "this item will get logged")
	logger.Log(logr.Error, "XXX - won't be logged as Error was not added to custom filter.")
	logger.Debug("XXX - won't be logged")
	logger.Log(LogoutLevel, "will get logged")

	err := lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}

	output := buf.String()
	fmt.Println(output)

	if !strings.Contains(output, "will get logged") {
		t.Error("missing levels")
	}

	if strings.Contains(output, "XXX") {
		t.Error("wrong level(s) output")
	}

}
