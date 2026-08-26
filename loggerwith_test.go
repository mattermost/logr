package logr_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoggerWith_ChainedCalls tests multiple With() calls in sequence.
func TestLoggerWith_ChainedCalls(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger = logger.With(logr.String("user", "alice"))
	logger = logger.With(logr.String("role", "admin"))
	logger = logger.With(logr.Int("session", 123))

	logger.Error("chained fields test")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "admin")
	assert.Contains(t, output, "123")
}

// TestLoggerWith_FieldOverride tests field override with same key.
func TestLoggerWith_FieldOverride(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger = logger.With(logr.String("key", "first"))
	logger = logger.With(logr.String("key", "second"))

	logger.Error("override test")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	// Both values should appear (fields are accumulated, not replaced)
	occurrences := strings.Count(output, "\"key\"")
	assert.Equal(t, 2, occurrences, "Both field values should be present")
}

// TestLoggerWith_DeepNesting tests deeply nested With() calls.
func TestLoggerWith_DeepNesting(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Create 20 nested loggers
	for i := 0; i < 20; i++ {
		logger = logger.With(logr.Int("level", i))
	}

	logger.Error("deep nesting")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "deep nesting")
	// Should have many level fields
	assert.Contains(t, output, "level")
}

// TestLoggerWith_EmptyFields tests With() with no fields.
func TestLoggerWith_EmptyFields(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger = logger.With() // No fields

	logger.Error("empty with")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "empty with")
}

// TestLoggerWith_FieldAccumulation tests that fields accumulate correctly.
func TestLoggerWith_FieldAccumulation(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	baseLogger := lgr.NewLogger()
	baseLogger = baseLogger.With(logr.String("app", "myapp"))

	// Branch 1
	logger1 := baseLogger.With(logr.String("module", "auth"))
	logger1.Error("auth message")

	// Branch 2
	logger2 := baseLogger.With(logr.String("module", "database"))
	logger2.Error("db message")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()

	// Both should have "app" field
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "auth message") {
			assert.Contains(t, line, "myapp")
			assert.Contains(t, line, "auth")
		}
		if strings.Contains(line, "db message") {
			assert.Contains(t, line, "myapp")
			assert.Contains(t, line, "database")
		}
	}
}

// TestLoggerWith_AllFieldTypes tests With() with various field types.
func TestLoggerWith_AllFieldTypes(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger = logger.With(
		logr.String("str", "value"),
		logr.Int("int", 42),
		logr.Int64("int64", 9223372036854775807),
		logr.Bool("bool", true),
		logr.Float32("float32", 3.14),
		logr.Float64("float64", 2.718),
	)

	logger.Error("all types")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "42")
	assert.Contains(t, output, "true")
	assert.Contains(t, output, "3.14")
}

// TestLoggerWith_MixedWithLogFields tests With() combined with fields in Log() call.
func TestLoggerWith_MixedWithLogFields(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger = logger.With(logr.String("context", "base"))

	// Add more fields in the log call
	logger.Error("mixed fields", logr.String("extra", "log-time"))

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "base")
	assert.Contains(t, output, "log-time")
}

// TestLoggerWith_Independence tests that loggers are independent.
func TestLoggerWith_Independence(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger1 := lgr.NewLogger().With(logr.String("id", "logger1"))
	logger2 := lgr.NewLogger().With(logr.String("id", "logger2"))

	logger1.Error("from logger1")
	logger2.Error("from logger2")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Each message should have only its own ID
	for _, line := range lines {
		if strings.Contains(line, "from logger1") {
			assert.Contains(t, line, "logger1")
			assert.NotContains(t, line, "logger2")
		}
		if strings.Contains(line, "from logger2") {
			assert.Contains(t, line, "logger2")
			assert.NotContains(t, line, "logger1")
		}
	}
}
