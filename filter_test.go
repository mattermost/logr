package logr_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStdFilter_AllLevels tests StdFilter with all standard levels.
func TestStdFilter_AllLevels(t *testing.T) {
	levels := []logr.Level{logr.Panic, logr.Fatal, logr.Error, logr.Warn, logr.Info, logr.Debug, logr.Trace}

	for _, filterLevel := range levels {
		t.Run("Filter_"+filterLevel.Name, func(t *testing.T) {
			filter := &logr.StdFilter{Lvl: filterLevel}

			for _, testLevel := range levels {
				level, enabled := filter.GetEnabledLevel(testLevel)
				if testLevel.ID <= filterLevel.ID {
					assert.True(t, enabled, "%s should be enabled with filter at %s", testLevel.Name, filterLevel.Name)
					// Check level properties (name, ID, color) but not stacktrace since it depends on filter config
					assert.Equal(t, testLevel.ID, level.ID, "Level ID should match")
					assert.Equal(t, testLevel.Name, level.Name, "Level name should match")
				} else {
					assert.False(t, enabled, "%s should be disabled with filter at %s", testLevel.Name, filterLevel.Name)
				}
			}
		})
	}
}

// TestStdFilter_StacktraceLevel tests stacktrace level configuration.
func TestStdFilter_StacktraceLevel(t *testing.T) {
	filter := &logr.StdFilter{Lvl: logr.Debug, Stacktrace: logr.Error}

	// Test levels above stacktrace threshold
	level, enabled := filter.GetEnabledLevel(logr.Error)
	assert.True(t, enabled)
	assert.True(t, level.Stacktrace, "Error should have stacktrace")

	level, enabled = filter.GetEnabledLevel(logr.Fatal)
	assert.True(t, enabled)
	assert.True(t, level.Stacktrace, "Fatal should have stacktrace")

	// Test levels below stacktrace threshold
	level, enabled = filter.GetEnabledLevel(logr.Warn)
	assert.True(t, enabled)
	assert.False(t, level.Stacktrace, "Warn should not have stacktrace")

	level, enabled = filter.GetEnabledLevel(logr.Info)
	assert.True(t, enabled)
	assert.False(t, level.Stacktrace, "Info should not have stacktrace")
}

// TestCustomFilter_EmptyFilter tests CustomFilter with no levels added.
func TestCustomFilter_EmptyFilter(t *testing.T) {
	filter := &logr.CustomFilter{}

	// All levels should be disabled
	_, enabled := filter.GetEnabledLevel(logr.Error)
	assert.False(t, enabled)

	customLevel := logr.Level{ID: 100, Name: "custom"}
	_, enabled = filter.GetEnabledLevel(customLevel)
	assert.False(t, enabled)
}

// TestCustomFilter_DuplicateLevels tests adding same level multiple times.
func TestCustomFilter_DuplicateLevels(t *testing.T) {
	filter := &logr.CustomFilter{}
	customLevel := logr.Level{ID: 100, Name: "custom"}

	// Add same level twice
	filter.Add(customLevel)
	filter.Add(customLevel)

	// Should still work correctly
	_, enabled := filter.GetEnabledLevel(customLevel)
	assert.True(t, enabled)
}

// TestCustomFilter_ManyLevels tests CustomFilter with many custom levels.
func TestCustomFilter_ManyLevels(t *testing.T) {
	filter := &logr.CustomFilter{}

	// Add 50 custom levels
	levels := make([]logr.Level, 50)
	for i := 0; i < 50; i++ {
		levels[i] = logr.Level{ID: logr.LevelID(100 + i), Name: "level"}
		filter.Add(levels[i])
	}

	// All should be enabled
	for _, lvl := range levels {
		_, enabled := filter.GetEnabledLevel(lvl)
		assert.True(t, enabled, "Level %d should be enabled", lvl.ID)
	}

	// Other levels should be disabled
	otherLevel := logr.Level{ID: 200, Name: "other"}
	_, enabled := filter.GetEnabledLevel(otherLevel)
	assert.False(t, enabled)
}

// TestCustomFilter_StacktraceFlag tests stacktrace flag in custom levels.
func TestCustomFilter_StacktraceFlag(t *testing.T) {
	filter := &logr.CustomFilter{}

	withStacktrace := logr.Level{ID: 100, Name: "withStack", Stacktrace: true}
	withoutStacktrace := logr.Level{ID: 101, Name: "withoutStack", Stacktrace: false}

	filter.Add(withStacktrace, withoutStacktrace)

	level, enabled := filter.GetEnabledLevel(withStacktrace)
	assert.True(t, enabled)
	assert.True(t, level.Stacktrace)

	level, enabled = filter.GetEnabledLevel(withoutStacktrace)
	assert.True(t, enabled)
	assert.False(t, level.Stacktrace)
}

// TestFilter_IntegrationWithLogging tests filters in actual logging scenarios.
func TestFilter_IntegrationWithLogging(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Warn}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Should be logged (Warn and above)
	logger.Error("error message")
	logger.Warn("warn message")
	logger.Fatal("fatal message")

	// Should NOT be logged (below Warn)
	logger.Info("info message")
	logger.Debug("debug message")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "fatal message")
	assert.NotContains(t, output, "info message")
	assert.NotContains(t, output, "debug message")
}

// TestCustomFilter_IntegrationWithLogging tests CustomFilter in logging.
func TestCustomFilter_IntegrationWithLogging(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	level1 := logr.Level{ID: 100, Name: "audit"}
	level2 := logr.Level{ID: 101, Name: "security"}
	level3 := logr.Level{ID: 102, Name: "performance"}

	filter := &logr.CustomFilter{}
	filter.Add(level1, level2) // Only audit and security

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	logger.Log(level1, "audit log")
	logger.Log(level2, "security log")
	logger.Log(level3, "performance log") // Should be filtered

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "audit log")
	assert.Contains(t, output, "security log")
	assert.NotContains(t, output, "performance log")
}

// TestMixedFilters_DifferentTargets tests different filter types on different targets.
func TestMixedFilters_DifferentTargets(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	// Target 1: StdFilter
	stdFilter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target1 := targets.NewWriterTarget(buf1)
	err = lgr.AddTarget(target1, "std", stdFilter, formatter, 100)
	require.NoError(t, err)

	// Target 2: CustomFilter
	customLevel := logr.Level{ID: 100, Name: "custom"}
	customFilter := &logr.CustomFilter{}
	customFilter.Add(customLevel)
	target2 := targets.NewWriterTarget(buf2)
	err = lgr.AddTarget(target2, "custom", customFilter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	logger.Error("error log")
	logger.Log(customLevel, "custom log")
	logger.Info("info log")

	err = lgr.Flush()
	require.NoError(t, err)

	// Target 1 should have only error
	output1 := buf1.String()
	assert.Contains(t, output1, "error log")
	assert.NotContains(t, output1, "custom log")
	assert.NotContains(t, output1, "info log")

	// Target 2 should have only custom
	output2 := buf2.String()
	assert.Contains(t, output2, "custom log")
	assert.NotContains(t, output2, "error log")
	assert.NotContains(t, output2, "info log")
}
