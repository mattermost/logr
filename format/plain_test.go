package format_test

import (
	"testing"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"
)

func TestPlain(t *testing.T) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	formatter := &format.Plain{DisableTimestamp: true, DisableStacktrace: true, Delim: " | "}
	target := target.NewWriterTarget(filter, formatter, buf, 1000)
	lgr.AddTarget(target)

	logger := lgr.NewLogger().WithField("name", "wiggin")

	logger.Error("This is an error.")
	lgr.Flush()

	got := buf.String()
	want := "error | This is an error. | name=wiggin\n"

	if got != want {
		t.Errorf("expected: \"%s\";  got:\"%s\"", got, want)
	}

	err := lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}
}
