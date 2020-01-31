package format_test

import (
	"strings"
	"testing"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"
)

func TestPlain(t *testing.T) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Panic}
	formatter := &format.Plain{DisableStacktrace: true, Delim: " | "}
	target := target.NewWriterTarget(filter, formatter, buf, 1000)
	err := lgr.AddTarget(target)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().WithField("name", "wiggin")

	logger.Error("This is an error.")
	lgr.Flush()

	got := buf.String()
	want := "error | This is an error. | name=wiggin\n"

	if !strings.Contains(got, want) {
		t.Errorf("expected: \"%s\";  got:\"%s\"", want, got)
	}

	t.Log(got)

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}
}
