package format_test

import (
	"testing"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"

	"github.com/nsf/jsondiff"
)

func TestJSON(t *testing.T) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	formatter := &format.JSON{DisableTimestamp: true, DisableStacktrace: true}
	target := target.NewWriterTarget(filter, formatter, buf, 1000)
	err := lgr.AddTarget(target)
	if err != nil {
		t.Error(err)
	}

	logger := lgr.NewLogger().WithField("name", "wiggin")

	logger.Error("This is an error.")
	lgr.Flush()

	want := `{"level":"error","msg":"This is an error.","name":"wiggin"}`

	opts := jsondiff.DefaultConsoleOptions()
	diff, _ := jsondiff.Compare(buf.Bytes(), []byte(want), &opts)
	if diff != jsondiff.FullMatch {
		t.Error(diff)
	}

	err = lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}
}
