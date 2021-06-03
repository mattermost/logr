package targets_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
)

func ExampleWriter() {
	lgr, _ := logr.New()
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	t := targets.NewWriterTarget(buf)
	_ = lgr.AddTarget(t, "example", filter, formatter, 1000)

	logger := lgr.NewLogger().With(logr.String("name", "wiggin")).Sugar()

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
	plain := &formatters.Plain{Delim: " | "}
	writer(t, plain)
}

func TestWriterJSON(t *testing.T) {
	json := &formatters.JSON{}
	writer(t, json)
}

func writer(t *testing.T, formatter logr.Formatter) {
	lgr, _ := logr.New()
	buf := &test.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	target := targets.NewWriterTarget(buf)
	_ = lgr.AddTarget(target, "writerTest", filter, formatter, 1000)

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
