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
	lgr, _ := logr.New()
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Panic}
	formatter := &formatters.Plain{DisableStacktrace: true, Delim: " | "}
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
