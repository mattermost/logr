package formatters_test

import (
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/require"
)

func TestPlain(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, DisableStacktrace: true, Delim: " | "}

	lgr, _ := logr.New()
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Panic}
	target := targets.NewWriterTarget(buf)
	err := lgr.AddTarget(target, "plainTest", filter, formatter, 1000)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "wiggin"))

	logger.Error("This is an error.")
	lgr.Flush()

	got := buf.String()
	want := "error | This is an error. | name=wiggin\n"

	if !strings.Contains(got, want) {
		t.Errorf("expected: \"%s\";  got:\"%s\"", want, got)
	}

	t.Log(got)

	err = lgr.Shutdown()
	require.NoError(t, err)
}

func TestPlainColorCustom(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, DisableStacktrace: true, Delim: " | ", EnableColor: true}

	lgr, _ := logr.New()
	buf := &test.Buffer{}

	customLevel := logr.Level{ID: 1000, Name: "CUST", Stacktrace: false, Color: logr.Cyan}
	filter := &logr.CustomFilter{}
	filter.Add(customLevel)

	target := targets.NewWriterTarget(buf)
	err := lgr.AddTarget(target, "plainTestColor", filter, formatter, 1000)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "wiggin"))

	logger.Log(customLevel, "This is a custom level with color.")
	lgr.Flush()

	got := buf.String()
	want := "\u001b[36mCUST\u001b[0m | This is a custom level with color. | \u001b[36mname\u001b[0m=wiggin\n"

	if !strings.Contains(got, want) {
		t.Errorf("expected: \"%s\";  got:\"%s\"", want, got)
	}

	t.Log(got)

	err = lgr.Shutdown()
	require.NoError(t, err)
}

func TestPlainColorStd(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, DisableStacktrace: true, Delim: " | ", EnableColor: true}

	lgr, _ := logr.New()
	buf := &test.Buffer{}

	filter := &logr.StdFilter{Lvl: logr.Debug, Stacktrace: logr.Panic}

	target := targets.NewWriterTarget(buf)
	err := lgr.AddTarget(target, "plainTestColor2", filter, formatter, 1000)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "wiggin"))

	logger.Error("This is an error level with color.")
	lgr.Flush()

	got := buf.String()
	want := "\u001b[31merror\u001b[0m | This is an error level with color. | \u001b[31mname\u001b[0m=wiggin\n"

	if !strings.Contains(got, want) {
		t.Errorf("expected: \"%s\";  got:\"%s\"", want, got)
	}

	logger.Info("Some info text")
	logger.Warn("A warning")
	logger.Debug("Some debug text")

	lgr.Flush()
	got = buf.String()

	t.Log(got)

	err = lgr.Shutdown()
	require.NoError(t, err)
}
