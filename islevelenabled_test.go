package logr_test

import (
	"context"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsLevelEnabled_NoTargets tests behavior when no targets exist.
func TestIsLevelEnabled_NoTargets(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// All levels should be disabled with no targets
	assert.False(t, lgr.IsLevelEnabled(logr.Panic).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Fatal).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Error).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Warn).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Info).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Debug).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Trace).Enabled)
}

// TestIsLevelEnabled_LevelHierarchy tests that StdFilter respects level hierarchy.
func TestIsLevelEnabled_LevelHierarchy(t *testing.T) {
	tests := []struct {
		name           string
		filterLevel    logr.Level
		expectedLevels map[logr.Level]bool
	}{
		{
			name:        "Fatal - only Fatal and Panic",
			filterLevel: logr.Fatal,
			expectedLevels: map[logr.Level]bool{
				logr.Panic: true,
				logr.Fatal: true,
				logr.Error: false,
				logr.Warn:  false,
				logr.Info:  false,
				logr.Debug: false,
				logr.Trace: false,
			},
		},
		{
			name:        "Error - Fatal, Error, Panic",
			filterLevel: logr.Error,
			expectedLevels: map[logr.Level]bool{
				logr.Panic: true,
				logr.Fatal: true,
				logr.Error: true,
				logr.Warn:  false,
				logr.Info:  false,
				logr.Debug: false,
				logr.Trace: false,
			},
		},
		{
			name:        "Warn - includes Error and above",
			filterLevel: logr.Warn,
			expectedLevels: map[logr.Level]bool{
				logr.Panic: true,
				logr.Fatal: true,
				logr.Error: true,
				logr.Warn:  true,
				logr.Info:  false,
				logr.Debug: false,
				logr.Trace: false,
			},
		},
		{
			name:        "Info - excludes Debug and Trace",
			filterLevel: logr.Info,
			expectedLevels: map[logr.Level]bool{
				logr.Panic: true,
				logr.Fatal: true,
				logr.Error: true,
				logr.Warn:  true,
				logr.Info:  true,
				logr.Debug: false,
				logr.Trace: false,
			},
		},
		{
			name:        "Debug - excludes only Trace",
			filterLevel: logr.Debug,
			expectedLevels: map[logr.Level]bool{
				logr.Panic: true,
				logr.Fatal: true,
				logr.Error: true,
				logr.Warn:  true,
				logr.Info:  true,
				logr.Debug: true,
				logr.Trace: false,
			},
		},
		{
			name:        "Trace - all levels enabled",
			filterLevel: logr.Trace,
			expectedLevels: map[logr.Level]bool{
				logr.Panic: true,
				logr.Fatal: true,
				logr.Error: true,
				logr.Warn:  true,
				logr.Info:  true,
				logr.Debug: true,
				logr.Trace: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lgr, err := logr.New()
			require.NoError(t, err)
			defer func() { require.NoError(t, lgr.Shutdown()) }()

			filter := &logr.StdFilter{Lvl: tt.filterLevel}
			formatter := &formatters.Plain{Delim: " | "}
			target := targets.NewWriterTarget(nil)
			err = lgr.AddTarget(target, "test", filter, formatter, 100)
			require.NoError(t, err)

			for level, expected := range tt.expectedLevels {
				status := lgr.IsLevelEnabled(level)
				assert.Equal(t, expected, status.Enabled, "Level %s should be %v", level.Name, expected)
			}
		})
	}
}

// TestIsLevelEnabled_StacktraceFlag tests that Stacktrace flag is set correctly.
func TestIsLevelEnabled_StacktraceFlag(t *testing.T) {
	tests := []struct {
		name             string
		filterLevel      logr.Level
		stacktraceLevel  logr.Level
		testLevel        logr.Level
		expectEnabled    bool
		expectStacktrace bool
	}{
		{
			name:             "Error enabled, stacktrace at Error",
			filterLevel:      logr.Error,
			stacktraceLevel:  logr.Error,
			testLevel:        logr.Error,
			expectEnabled:    true,
			expectStacktrace: true,
		},
		{
			name:             "Warn enabled, stacktrace at Error",
			filterLevel:      logr.Warn,
			stacktraceLevel:  logr.Error,
			testLevel:        logr.Warn,
			expectEnabled:    true,
			expectStacktrace: false,
		},
		{
			name:             "Error level, stacktrace at Error",
			filterLevel:      logr.Warn,
			stacktraceLevel:  logr.Error,
			testLevel:        logr.Error,
			expectEnabled:    true,
			expectStacktrace: true,
		},
		{
			name:             "Panic always has stacktrace when enabled",
			filterLevel:      logr.Fatal,
			stacktraceLevel:  logr.Error,
			testLevel:        logr.Panic,
			expectEnabled:    true,
			expectStacktrace: true,
		},
		{
			name:             "Disabled level has no stacktrace",
			filterLevel:      logr.Error,
			stacktraceLevel:  logr.Error,
			testLevel:        logr.Info,
			expectEnabled:    false,
			expectStacktrace: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lgr, err := logr.New()
			require.NoError(t, err)
			defer func() { require.NoError(t, lgr.Shutdown()) }()

			filter := &logr.StdFilter{Lvl: tt.filterLevel, Stacktrace: tt.stacktraceLevel}
			formatter := &formatters.Plain{Delim: " | "}
			target := targets.NewWriterTarget(nil)
			err = lgr.AddTarget(target, "test", filter, formatter, 100)
			require.NoError(t, err)

			status := lgr.IsLevelEnabled(tt.testLevel)
			assert.Equal(t, tt.expectEnabled, status.Enabled, "Enabled mismatch")
			assert.Equal(t, tt.expectStacktrace, status.Stacktrace, "Stacktrace mismatch")
		})
	}
}

// TestIsLevelEnabled_MultipleTargets tests level checking with multiple targets.
func TestIsLevelEnabled_MultipleTargets(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Target 1: Error and above
	filter1 := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target1 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target1, "target1", filter1, formatter, 100)
	require.NoError(t, err)

	// Target 2: Debug and above
	filter2 := &logr.StdFilter{Lvl: logr.Debug}
	target2 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target2, "target2", filter2, formatter, 100)
	require.NoError(t, err)

	// Error should be enabled (both targets)
	assert.True(t, lgr.IsLevelEnabled(logr.Error).Enabled)

	// Debug should be enabled (target2)
	assert.True(t, lgr.IsLevelEnabled(logr.Debug).Enabled)

	// Trace should be disabled (no target)
	assert.False(t, lgr.IsLevelEnabled(logr.Trace).Enabled)
}

// TestIsLevelEnabled_CustomLevels tests IsLevelEnabled with custom levels.
func TestIsLevelEnabled_CustomLevels(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	customLevel1 := logr.Level{ID: 100, Name: "custom1"}
	customLevel2 := logr.Level{ID: 101, Name: "custom2"}
	customLevel3 := logr.Level{ID: 102, Name: "custom3"}

	filter := &logr.CustomFilter{}
	filter.Add(customLevel1, customLevel2)

	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// customLevel1 and customLevel2 should be enabled
	assert.True(t, lgr.IsLevelEnabled(customLevel1).Enabled)
	assert.True(t, lgr.IsLevelEnabled(customLevel2).Enabled)

	// customLevel3 should be disabled
	assert.False(t, lgr.IsLevelEnabled(customLevel3).Enabled)

	// Standard levels should be disabled
	assert.False(t, lgr.IsLevelEnabled(logr.Error).Enabled)
}

// TestIsLevelEnabled_MixedFilters tests with both StdFilter and CustomFilter targets.
func TestIsLevelEnabled_MixedFilters(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Target 1: StdFilter at Error
	filter1 := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target1 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target1, "std", filter1, formatter, 100)
	require.NoError(t, err)

	// Target 2: CustomFilter with custom level
	customLevel := logr.Level{ID: 100, Name: "custom"}
	filter2 := &logr.CustomFilter{}
	filter2.Add(customLevel)
	target2 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target2, "custom", filter2, formatter, 100)
	require.NoError(t, err)

	// Error should be enabled (StdFilter target)
	assert.True(t, lgr.IsLevelEnabled(logr.Error).Enabled)

	// Custom level should be enabled (CustomFilter target)
	assert.True(t, lgr.IsLevelEnabled(customLevel).Enabled)

	// Info should be disabled (not in StdFilter at Error level)
	assert.False(t, lgr.IsLevelEnabled(logr.Info).Enabled)
}

// TestIsLevelEnabled_AfterShutdown tests that IsLevelEnabled returns false after shutdown.
func TestIsLevelEnabled_AfterShutdown(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// Error should be enabled before shutdown
	assert.True(t, lgr.IsLevelEnabled(logr.Error).Enabled)

	err = lgr.Shutdown()
	require.NoError(t, err)

	// All levels should be disabled after shutdown
	assert.False(t, lgr.IsLevelEnabled(logr.Error).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Info).Enabled)
	assert.False(t, lgr.IsLevelEnabled(logr.Debug).Enabled)
}

// TestIsLevelEnabled_CacheInvalidationOnRemove tests cache is cleared when targets removed.
func TestIsLevelEnabled_CacheInvalidationOnRemove(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Debug}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// Prime cache - Debug should be enabled
	assert.True(t, lgr.IsLevelEnabled(logr.Debug).Enabled)

	// Remove target
	err = lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
		return ti.Name == "test"
	})
	require.NoError(t, err)

	// Cache should be invalidated - Debug should now be disabled
	assert.False(t, lgr.IsLevelEnabled(logr.Debug).Enabled)
}

// TestIsLevelEnabled_StacktraceMultipleTargets tests stacktrace flag with multiple targets.
func TestIsLevelEnabled_StacktraceMultipleTargets(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	formatter := &formatters.Plain{Delim: " | "}

	// Target 1: Error level, no stacktrace
	filter1 := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Panic}
	target1 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target1, "target1", filter1, formatter, 100)
	require.NoError(t, err)

	// Target 2: Error level with stacktrace at Error
	filter2 := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	target2 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target2, "target2", filter2, formatter, 100)
	require.NoError(t, err)

	// Error should be enabled with stacktrace (target2 requests it)
	status := lgr.IsLevelEnabled(logr.Error)
	assert.True(t, status.Enabled)
	assert.True(t, status.Stacktrace, "Stacktrace should be true if ANY target requests it")
}

// TestIsLevelEnabled_CustomLevelBoundaries tests custom levels at ID boundaries.
func TestIsLevelEnabled_CustomLevelBoundaries(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Test with level IDs at the boundaries
	minCustom := logr.Level{ID: 11, Name: "minCustom"} // Just above standard levels
	midCustom := logr.Level{ID: 1000, Name: "midCustom"}
	maxCustom := logr.Level{ID: logr.MaxLevelID, Name: "maxCustom"}

	filter := &logr.CustomFilter{}
	filter.Add(minCustom, midCustom, maxCustom)

	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// All three custom levels should be enabled
	assert.True(t, lgr.IsLevelEnabled(minCustom).Enabled)
	assert.True(t, lgr.IsLevelEnabled(midCustom).Enabled)
	assert.True(t, lgr.IsLevelEnabled(maxCustom).Enabled)

	// Level IDs outside the added ones should be disabled
	outsideLevel := logr.Level{ID: 500, Name: "outside"}
	assert.False(t, lgr.IsLevelEnabled(outsideLevel).Enabled)
}
