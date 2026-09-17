package formatters_test

import (
	"encoding/json"
	"errors"
	"sort"
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

// gelfOutput represents the GELF JSON structure for testing
type gelfOutput struct {
	Version      string  `json:"version"`
	Host         string  `json:"host"`
	ShortMessage string  `json:"short_message"`
	FullMessage  string  `json:"full_message,omitempty"`
	Timestamp    float64 `json:"timestamp"`
	Level        uint32  `json:"level"`
	Caller       string  `json:"_caller,omitempty"`
}

func TestGelfCheckValid(t *testing.T) {
	gelf := &formatters.Gelf{}
	err := gelf.CheckValid()
	require.NoError(t, err, "CheckValid should always return nil")
}

func TestGelfIsStacktraceNeeded(t *testing.T) {
	t.Run("EnableCaller true", func(t *testing.T) {
		gelf := &formatters.Gelf{EnableCaller: true}
		assert.True(t, gelf.IsStacktraceNeeded())
	})

	t.Run("EnableCaller false", func(t *testing.T) {
		gelf := &formatters.Gelf{EnableCaller: false}
		assert.False(t, gelf.IsStacktraceNeeded())
	})
}

func TestGelfBasicFormat(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	formatter := &formatters.Gelf{}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("Test message")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	// Remove null terminator
	output = strings.TrimRight(output, "\x00\n")

	var gelfMsg gelfOutput
	err = json.Unmarshal([]byte(output), &gelfMsg)
	require.NoError(t, err, "Should produce valid JSON")

	assert.Equal(t, "1.1", gelfMsg.Version)
	assert.Equal(t, "Test message", gelfMsg.ShortMessage)
	assert.NotEmpty(t, gelfMsg.Host)
	assert.Greater(t, gelfMsg.Timestamp, float64(0))
	assert.Equal(t, uint32(logr.Info.ID), gelfMsg.Level)
}

func TestGelfEmptyMessage(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info}
	formatter := &formatters.Gelf{}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("") // Empty message

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var gelfMsg gelfOutput
	err = json.Unmarshal([]byte(output), &gelfMsg)
	require.NoError(t, err)

	// GELF requires non-empty short_message, so "-" is used as fallback
	assert.Equal(t, "-", gelfMsg.ShortMessage)
}

func TestGelfCustomHostname(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info}
	customHost := "custom.host.example.com"
	formatter := &formatters.Gelf{Hostname: customHost}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("Test")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var gelfMsg gelfOutput
	err = json.Unmarshal([]byte(output), &gelfMsg)
	require.NoError(t, err)

	assert.Equal(t, customHost, gelfMsg.Host)
}

func TestGelfWithFields(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info}
	formatter := &formatters.Gelf{}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("Test with fields",
		logr.String("string_field", "value"),
		logr.Int("int_field", 42),
		logr.Bool("bool_field", true),
		logr.Float("float_field", 3.14),
	)

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	// GELF custom fields should have underscore prefix
	assert.Equal(t, "value", result["_string_field"])
	assert.Equal(t, float64(42), result["_int_field"])
	assert.Equal(t, true, result["_bool_field"])
	assert.Equal(t, 3.14, result["_float_field"])
}

func TestGelfWithCaller(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Need to enable stacktrace in filter for caller info to be available
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Info}
	formatter := &formatters.Gelf{EnableCaller: true}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("Test with caller")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	caller, ok := result["_caller"].(string)
	assert.True(t, ok, "Should have _caller field")
	assert.Contains(t, caller, "gelf_test.go")
}

func TestGelfWithStacktrace(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	formatter := &formatters.Gelf{}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Error("Error with stacktrace")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var gelfMsg gelfOutput
	err = json.Unmarshal([]byte(output), &gelfMsg)
	require.NoError(t, err)

	// full_message should contain stacktrace
	assert.NotEmpty(t, gelfMsg.FullMessage)
	assert.Contains(t, gelfMsg.FullMessage, "gelf_test.go")
}

func TestGelfFieldSorter(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info}
	sorter := func(fields []logr.Field) []logr.Field {
		cf := make([]logr.Field, len(fields))
		copy(cf, fields)
		sort.Sort(logr.FieldSorter(cf))
		return cf
	}
	formatter := &formatters.Gelf{FieldSorter: sorter}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.Info("Test sorting",
		logr.String("zebra", "last"),
		logr.String("alpha", "first"),
		logr.String("middle", "second"),
	)

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	// Fields should be present (sorting affects iteration order in encoder)
	assert.Contains(t, output, "_zebra")
	assert.Contains(t, output, "_alpha")
	assert.Contains(t, output, "_middle")
}

func TestGelfComplexTypes(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info}
	formatter := &formatters.Gelf{}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	now := time.Now()
	dur := time.Hour + time.Minute*30

	logger.Info("Complex types",
		logr.Time("time_field", now),
		logr.Duration("duration_field", dur),
		logr.Err(errors.New("test error")),
	)

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var result map[string]interface{}
	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err)

	assert.NotNil(t, result["_time_field"])
	assert.NotNil(t, result["_duration_field"])
	assert.Contains(t, result["_error"], "test error")
}

func TestGelfDifferentLogLevels(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	levels := []struct {
		level    logr.Level
		expected uint32
	}{
		{logr.Panic, uint32(logr.Panic.ID)},
		{logr.Fatal, uint32(logr.Fatal.ID)},
		{logr.Error, uint32(logr.Error.ID)},
		{logr.Warn, uint32(logr.Warn.ID)},
		{logr.Info, uint32(logr.Info.ID)},
		{logr.Debug, uint32(logr.Debug.ID)},
		{logr.Trace, uint32(logr.Trace.ID)},
	}

	for _, tc := range levels {
		t.Run(tc.level.Name, func(t *testing.T) {
			filter := &logr.StdFilter{Lvl: tc.level}
			formatter := &formatters.Gelf{}

			buf := &test.Buffer{}
			target := targets.NewWriterTarget(buf)
			err = lgr.AddTarget(target, "gelfTest_"+tc.level.Name, filter, formatter, 1000)
			require.NoError(t, err)

			logger := lgr.NewLogger()
			logger.Log(tc.level, "Test "+tc.level.Name)

			err = lgr.Flush()
			require.NoError(t, err)

			output := buf.String()
			output = strings.TrimRight(output, "\x00\n")

			var gelfMsg gelfOutput
			err = json.Unmarshal([]byte(output), &gelfMsg)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, gelfMsg.Level)
		})
	}
}

func TestGelfTimestamp(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Info}
	formatter := &formatters.Gelf{}

	buf := &test.Buffer{}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "gelfTest", filter, formatter, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	beforeTime := time.Now().Unix()
	logger.Info("Timestamp test")
	afterTime := time.Now().Unix()

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	output = strings.TrimRight(output, "\x00\n")

	var gelfMsg gelfOutput
	err = json.Unmarshal([]byte(output), &gelfMsg)
	require.NoError(t, err)

	// Timestamp should be within reasonable range (seconds + milliseconds)
	logTime := int64(gelfMsg.Timestamp)
	assert.GreaterOrEqual(t, logTime, beforeTime)
	assert.LessOrEqual(t, logTime, afterTime+1) // Allow 1 second margin

	// Should have millisecond precision (fractional part)
	assert.Greater(t, gelfMsg.Timestamp, float64(logTime))
}
