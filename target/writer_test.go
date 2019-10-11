package target_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/logr/format"
	"github.com/wiggin77/logr/target"
	"github.com/wiggin77/logr/test"
)

func ExampleWriter() {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.Plain{Delim: " | "}
	t := target.NewWriterTarget(filter, formatter, buf, 1000)
	lgr.AddTarget(t)

	logger := lgr.NewLogger().WithField("name", "wiggin")

	logger.Errorf("the erroneous data is %s", test.StringRnd(10))
	logger.Warnf("strange data: %s", test.StringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	err := lgr.Shutdown()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	output := buf.String()
	fmt.Println(output)
}

func TestWriterPlain(t *testing.T) {
	plain := &format.Plain{Delim: " | "}
	writer(t, plain)
}

func TestWriterJSON(t *testing.T) {
	json := &format.JSON{Indent: "  "}
	writer(t, json)
}

func writer(t *testing.T, formatter logr.Formatter) {
	lgr := &logr.Logr{}
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	target := target.NewWriterTarget(filter, formatter, buf, 1000)
	lgr.AddTarget(target)

	const goodToken = "Woot!"
	const badToken = "XXX!!XXX"

	cfg := test.DoSomeLoggingCfg{
		Lgr:        lgr,
		Goroutines: 10,
		Loops:      50,
		GoodToken:  goodToken,
		BadToken:   badToken,
		Lvl:        logr.Error,
		Delay:      time.Millisecond * 1,
	}
	test.DoSomeLogging(cfg)
	err := lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}

	output := buf.String()
	fmt.Println(output)

	if !strings.Contains(output, goodToken) {
		t.Errorf("missing warnings")
	}

	if strings.Contains(output, badToken) {
		t.Errorf("wrong level(s) enabled")
	}
}
