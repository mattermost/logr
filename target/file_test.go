package target_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/format"
	"github.com/mattermost/logr/v2/target"
	"github.com/mattermost/logr/v2/test"
)

func ExampleFile() {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Warn, Stacktrace: logr.Error}
	formatter := &format.JSON{}
	opts := target.FileOptions{
		Filename:   "./logs/test_lumberjack.log",
		MaxSize:    1,
		MaxAge:     2,
		MaxBackups: 3,
		Compress:   false,
	}
	t := target.NewFileTarget(opts)
	_ = lgr.AddTarget(t, "test", filter, formatter, 1000)

	logger := lgr.NewLogger().WithField("name", "wiggin")

	logger.Errorf("the erroneous data is %s", test.StringRnd(10))
	logger.Warnf("strange data: %s", test.StringRnd(5))
	logger.Debug("XXX")
	logger.Trace("XXX")

	err := lgr.Shutdown()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func TestFilePlain(t *testing.T) {
	plain := &format.Plain{Delim: " | "}
	file(t, plain, "./logs/test_lumberjack_plain.log")
}

func TestFileJSON(t *testing.T) {
	json := &format.JSON{Indent: "\n  "}
	file(t, json, "./logs/test_lumberjack_json.log")
}

func file(t *testing.T, formatter logr.Formatter, filename string) {
	lgr, _ := logr.New()

	fileOpts := target.FileOptions{
		Filename:   filename,
		MaxSize:    1,
		MaxAge:     2,
		MaxBackups: 3,
		Compress:   false,
	}

	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	tgt := target.NewFileTarget(fileOpts)
	_ = lgr.AddTarget(tgt, "test2", filter, formatter, 1000)

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

	if !fileContains(t, filename, goodToken) {
		t.Errorf("missing warnings")
	}

	if fileContains(t, filename, badToken) {
		t.Errorf("wrong level(s) enabled")
	}
}

func fileContains(t *testing.T, filename string, text string) bool {
	file, err := os.Open(filename)
	if err != nil {
		t.Error(err)
		return false
	}
	defer file.Close()

	const bufSize = 1000 * 1024
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, bufSize), bufSize)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), text) {
			return true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Error(err)
	}
	return false
}
