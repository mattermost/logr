package logr_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogM_Basic tests basic multi-level logging functionality.
func TestLogM_Basic(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}
	filter := &logr.StdFilter{Lvl: logr.Warn}
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// Create custom levels to simulate tags
	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}
	tag3 := logr.Level{ID: 102, Name: "tag3"}

	logger := lgr.NewLogger()

	// Log with multiple levels
	logger.LogM([]logr.Level{tag1, tag2, tag3}, "multi-level log entry")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	// Should not be logged since custom levels not in StdFilter
	assert.Empty(t, output)
}

// TestLogM_WithCustomFilter tests LogM with CustomFilter that enables the levels.
func TestLogM_WithCustomFilter(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}

	filter := &logr.CustomFilter{}
	filter.Add(tag1, tag2)

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.LogM([]logr.Level{tag1, tag2}, "should be logged")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "should be logged")
}

// TestLogM_EmptyLevelSlice tests LogM with empty level slice.
func TestLogM_EmptyLevelSlice(t *testing.T) {
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

	// Empty slice should not panic, should be no-op
	logger.LogM([]logr.Level{}, "should not be logged")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Empty(t, output)
}

// TestLogM_SingleLevel tests LogM with single level (equivalent to Log).
func TestLogM_SingleLevel(t *testing.T) {
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
	logger.LogM([]logr.Level{logr.Error}, "single level")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "single level")
}

// TestLogM_ManyLevels tests LogM with many levels.
func TestLogM_ManyLevels(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	// Create many custom levels
	levels := make([]logr.Level, 10)
	filter := &logr.CustomFilter{}
	for i := 0; i < 10; i++ {
		levels[i] = logr.Level{ID: logr.LevelID(100 + i), Name: "tag"}
		filter.Add(levels[i])
	}

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.LogM(levels, "many levels")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "many levels")
}

// TestLogM_MixedFilters tests LogM with multiple targets having different filters.
func TestLogM_MixedFilters(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}
	tag3 := logr.Level{ID: 102, Name: "tag3"}

	// Target 1: only tag1 and tag2
	filter1 := &logr.CustomFilter{}
	filter1.Add(tag1, tag2)
	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target1 := targets.NewWriterTarget(buf1)
	err = lgr.AddTarget(target1, "target1", filter1, formatter, 100)
	require.NoError(t, err)

	// Target 2: only tag2 and tag3
	filter2 := &logr.CustomFilter{}
	filter2.Add(tag2, tag3)
	target2 := targets.NewWriterTarget(buf2)
	err = lgr.AddTarget(target2, "target2", filter2, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Log with tag1 and tag2 - should go to target1 only
	logger.LogM([]logr.Level{tag1, tag2}, "entry1")

	// Log with tag2 and tag3 - should go to target2 only
	logger.LogM([]logr.Level{tag2, tag3}, "entry2")

	// Log with all three - should go to both targets
	logger.LogM([]logr.Level{tag1, tag2, tag3}, "entry3")

	err = lgr.Flush()
	require.NoError(t, err)

	output1 := buf1.String()
	output2 := buf2.String()

	// LogM logs once per tag, and each tag is filtered independently
	// entry1: logged with tag1 and tag2
	//   - tag1 log goes to target1 (has tag1)
	//   - tag2 log goes to both targets (both have tag2)
	// entry2: logged with tag2 and tag3
	//   - tag2 log goes to both targets (both have tag2)
	//   - tag3 log goes to target2 (has tag3)
	// entry3: logged with tag1, tag2, and tag3
	//   - tag1 log goes to target1 (has tag1)
	//   - tag2 log goes to both targets (both have tag2)
	//   - tag3 log goes to target2 (has tag3)

	// Both targets should have all entries because tag2 is common
	assert.Contains(t, output1, "entry1")
	assert.Contains(t, output1, "entry2")
	assert.Contains(t, output1, "entry3")

	assert.Contains(t, output2, "entry1")
	assert.Contains(t, output2, "entry2")
	assert.Contains(t, output2, "entry3")
}

// TestLogM_AllDisabled tests LogM when all levels are disabled.
func TestLogM_AllDisabled(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	// Create custom levels but don't add to filter
	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}

	filter := &logr.CustomFilter{}
	// Don't add any levels

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()
	logger.LogM([]logr.Level{tag1, tag2}, "should not be logged")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Empty(t, output)
}

// TestLogM_CacheBehavior tests that LogM properly caches level checks.
func TestLogM_CacheBehavior(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	tag1 := logr.Level{ID: 100, Name: "tag1"}
	tag2 := logr.Level{ID: 101, Name: "tag2"}

	filter := &logr.CustomFilter{}
	filter.Add(tag1, tag2)

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// First call - populates cache
	logger.LogM([]logr.Level{tag1, tag2}, "first")

	// Second call - should hit cache
	logger.LogM([]logr.Level{tag1, tag2}, "second")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "first")
	assert.Contains(t, output, "second")

	// Count occurrences - LogM logs once per level, so 2 levels * 2 calls = 4 lines
	count := strings.Count(output, "\n")
	assert.Equal(t, 4, count, "Should have 4 log lines (2 levels * 2 messages)")
}

// TestLogM_WithFields tests LogM with contextual fields.
func TestLogM_WithFields(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	tag1 := logr.Level{ID: 100, Name: "tag1"}

	filter := &logr.CustomFilter{}
	filter.Add(tag1)

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger().With(logr.String("user", "alice"))
	logger.LogM([]logr.Level{tag1}, "with fields", logr.Int("count", 42))

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "with fields")
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "42")
}

// TestLogM_AfterRemoveTarget tests LogM cache invalidation after target removal.
func TestLogM_AfterRemoveTarget(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	buf := &bytes.Buffer{}

	tag1 := logr.Level{ID: 100, Name: "tag1"}

	filter := &logr.CustomFilter{}
	filter.Add(tag1)

	formatter := &formatters.Plain{Delim: " | ", DisableTimestamp: true}
	target := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// Prime cache
	logger.LogM([]logr.Level{tag1}, "before remove")

	// Flush to ensure "before remove" is written to buffer
	err = lgr.Flush()
	require.NoError(t, err)

	// Remove target
	err = lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
		return ti.Name == "test"
	})
	require.NoError(t, err)

	// Should not be logged now (cache should be invalidated)
	logger.LogM([]logr.Level{tag1}, "after remove")

	err = lgr.Flush()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "before remove")
	assert.NotContains(t, output, "after remove")
}
