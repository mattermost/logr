package formatters_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/assert"
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

func TestPlainTimestamp(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}

	t.Run("main timestamp uses local time", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)

		// Create formatter with timestamps enabled
		timestampFormatter := &formatters.Plain{
			DisableStacktrace: true,
			Delim:             " | ",
		}

		err := lgr.AddTarget(target, "timestampTest", filter, timestampFormatter, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger()

		// Record the time before logging
		beforeLog := time.Now()
		logger.Error("Timestamp test")

		err = lgr.Flush()
		require.NoError(t, err)

		// Record the time after logging
		afterLog := time.Now()

		output := buf.String()

		// Extract timestamp from plain output - format: [2006-01-02 15:04:05.000 Z07:00]
		timestampRegex := `\[([^\]]+)\]`
		re := regexp.MustCompile(timestampRegex)
		matches := re.FindStringSubmatch(output)
		if len(matches) < 2 {
			t.Fatalf("Could not find timestamp in output: %s", output)
		}

		timestampStr := matches[1]

		// Parse the timestamp from the log output
		loggedTime, err := time.Parse(logr.DefTimestampFormat, timestampStr)
		require.NoErrorf(t, err, "Could not parse timestamp %s: %v", timestampStr, err)

		if isUTC() {
			// Special case for systems that use UTC as their local time
			assert.Equal(t, loggedTime.Location(), time.UTC)
		} else {
			// Verify the logged time uses local timezone (not UTC)
			assert.NotEqual(t, loggedTime.Location(), time.UTC, "Expected logged time to use local timezone, but got UTC")
		}

		// Check that the logged time is between our before/after markers (allowing 1 second buffer)
		if loggedTime.Before(beforeLog.Add(-time.Second)) || loggedTime.After(afterLog.Add(time.Second)) {
			t.Errorf("Logged time %v is not between expected range %v to %v", loggedTime, beforeLog, afterLog)
		}
	})

	t.Run("UseUTC converts timestamp to UTC", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)

		// Create formatter with UseUTC enabled
		utcFormatter := &formatters.Plain{
			DisableStacktrace: true,
			Delim:             " | ",
			UseUTC:            true,
		}

		err := lgr.AddTarget(target, "utcTest", filter, utcFormatter, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger()

		// Record the time before logging
		beforeLog := time.Now()
		logger.Error("UTC test")

		err = lgr.Flush()
		require.NoError(t, err)

		// Record the time after logging
		afterLog := time.Now()

		output := buf.String()

		// Extract timestamp from plain output - format: [2006-01-02 15:04:05.000 Z07:00]
		timestampRegex := `\[([^\]]+)\]`
		re := regexp.MustCompile(timestampRegex)
		matches := re.FindStringSubmatch(output)
		if len(matches) < 2 {
			t.Fatalf("Could not find timestamp in output: %s", output)
		}

		timestampStr := matches[1]

		// Parse the timestamp from the log output using UTC format (no timezone)
		loggedTime, err := time.Parse(logr.DefTimestampFormat, timestampStr)
		require.NoErrorf(t, err, "Could not parse timestamp %s: %v", timestampStr, err)

		// Verify the logged time is in UTC timezone and format excludes timezone suffix
		assert.Equal(t, loggedTime.Location(), time.UTC, "Expected logged time to be in UTC timezone")

		// Check that the logged time is between our before/after markers (allowing 1 second buffer)
		// Convert to UTC for comparison since the logged time is in UTC
		beforeLogUTC := beforeLog.UTC()
		afterLogUTC := afterLog.UTC()
		if loggedTime.Before(beforeLogUTC.Add(-time.Second)) || loggedTime.After(afterLogUTC.Add(time.Second)) {
			t.Errorf("Logged time %v is not between expected UTC range %v to %v", loggedTime, beforeLogUTC, afterLogUTC)
		}
	})

	err = lgr.Shutdown()
	require.NoError(t, err)
}
