package logr_test

import (
	"context"
	"sync"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCacheInvalidationOnAddTarget verifies that the level cache is properly
// invalidated when a new target is added.
func TestCacheInvalidationOnAddTarget(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Initially, Error level should not be enabled (no targets)
	status := lgr.IsLevelEnabled(logr.Error)
	assert.False(t, status.Enabled, "Error level should not be enabled with no targets")

	// Add a target that enables Error level
	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// Now Error level should be enabled (cache invalidated and repopulated)
	status = lgr.IsLevelEnabled(logr.Error)
	assert.True(t, status.Enabled, "Error level should be enabled after adding target")

	// Debug should still be disabled
	status = lgr.IsLevelEnabled(logr.Debug)
	assert.False(t, status.Enabled, "Debug level should remain disabled")
}

// TestCacheInvalidationOnRemoveTarget verifies that the level cache is properly
// invalidated when targets are removed.
func TestCacheInvalidationOnRemoveTarget(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Add two targets with different filters
	filter1 := &logr.StdFilter{Lvl: logr.Error}
	filter2 := &logr.StdFilter{Lvl: logr.Warn}
	formatter := &formatters.Plain{Delim: " | "}

	target1 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target1, "target1", filter1, formatter, 100)
	require.NoError(t, err)

	target2 := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target2, "target2", filter2, formatter, 100)
	require.NoError(t, err)

	// Both Warn and Error should be enabled
	assert.True(t, lgr.IsLevelEnabled(logr.Warn).Enabled)
	assert.True(t, lgr.IsLevelEnabled(logr.Error).Enabled)

	// Remove target2 (which enabled Warn)
	err = lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
		return ti.Name == "target2"
	})
	require.NoError(t, err)

	// Warn should now be disabled (only Error remains)
	assert.False(t, lgr.IsLevelEnabled(logr.Warn).Enabled, "Warn should be disabled after removing target2")
	assert.True(t, lgr.IsLevelEnabled(logr.Error).Enabled, "Error should still be enabled")
}

// TestResetLevelCacheConcurrent tests that ResetLevelCache can be safely called
// concurrently with IsLevelEnabled checks.
func TestResetLevelCacheConcurrent(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// Prime the cache
	lgr.IsLevelEnabled(logr.Error)

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutines continuously checking levels
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					lgr.IsLevelEnabled(logr.Error)
					lgr.IsLevelEnabled(logr.Warn)
					lgr.IsLevelEnabled(logr.Debug)
				}
			}
		}()
	}

	// Goroutine continuously resetting cache
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			lgr.ResetLevelCache()
		}
	}()

	// Let them run for a bit
	// Note: No explicit sleep needed, the 100 iterations provide enough overlap

	close(stopCh)
	wg.Wait()

	// If we got here without deadlock or panic, the test passes
}

// TestPerTargetCacheWithCustomFilter tests that per-target caching works
// correctly with custom filters.
func TestPerTargetCacheWithCustomFilter(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	// Create custom levels
	customLevel1 := logr.Level{ID: 100, Name: "custom1"}
	customLevel2 := logr.Level{ID: 101, Name: "custom2"}

	// Add target with custom filter
	filter := &logr.CustomFilter{}
	filter.Add(customLevel1)
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	// These should work without error (testing per-target cache)
	logger.Log(customLevel1, "should be logged")
	logger.Log(customLevel2, "should be filtered")

	// Verify top-level cache works
	assert.True(t, lgr.IsLevelEnabled(customLevel1).Enabled)
	assert.False(t, lgr.IsLevelEnabled(customLevel2).Enabled)
}

// TestCacheOptionsBackwardCompatibility tests that the old and new cache
// options work correctly.
func TestCacheOptionsBackwardCompatibility(t *testing.T) {
	// Test default (should be syncMap)
	lgr1, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr1.Shutdown()) }()

	filter := &logr.StdFilter{Lvl: logr.Error}
	formatter := &formatters.Plain{Delim: " | "}
	target := targets.NewWriterTarget(nil)
	err = lgr1.AddTarget(target, "test", filter, formatter, 100)
	require.NoError(t, err)

	// Should work fine with default cache
	status := lgr1.IsLevelEnabled(logr.Error)
	assert.True(t, status.Enabled)

	// Test explicit array cache
	lgr2, err := logr.New(logr.UseArrayLevelCache(true))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr2.Shutdown()) }()

	target2 := targets.NewWriterTarget(nil)
	err = lgr2.AddTarget(target2, "test", filter, formatter, 100)
	require.NoError(t, err)

	status = lgr2.IsLevelEnabled(logr.Error)
	assert.True(t, status.Enabled)

	// Test deprecated UseSyncMapLevelCache(true) - should use syncMap
	lgr3, err := logr.New(logr.UseSyncMapLevelCache(true))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr3.Shutdown()) }()

	target3 := targets.NewWriterTarget(nil)
	err = lgr3.AddTarget(target3, "test", filter, formatter, 100)
	require.NoError(t, err)

	status = lgr3.IsLevelEnabled(logr.Error)
	assert.True(t, status.Enabled)

	// Test deprecated UseSyncMapLevelCache(false) - should use array
	lgr4, err := logr.New(logr.UseSyncMapLevelCache(false))
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr4.Shutdown()) }()

	target4 := targets.NewWriterTarget(nil)
	err = lgr4.AddTarget(target4, "test", filter, formatter, 100)
	require.NoError(t, err)

	status = lgr4.IsLevelEnabled(logr.Error)
	assert.True(t, status.Enabled)
}

// TestMultipleTargetsCacheInvalidation tests that per-target caches are
// properly cleared for all targets when cache is reset.
func TestMultipleTargetsCacheInvalidation(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	formatter := &formatters.Plain{Delim: " | "}

	// Add multiple targets with different filters
	for i := 0; i < 5; i++ {
		filter := &logr.StdFilter{Lvl: logr.Error}
		target := targets.NewWriterTarget(nil)
		err = lgr.AddTarget(target, "test", filter, formatter, 100)
		require.NoError(t, err)
	}

	// Prime all caches
	lgr.IsLevelEnabled(logr.Error)
	lgr.IsLevelEnabled(logr.Warn)

	logger := lgr.NewLogger()
	// This will check per-target caches
	logger.Error("test")
	logger.Warn("test")

	// Reset should clear all caches (top-level and per-target)
	lgr.ResetLevelCache()

	// Should still work correctly after reset
	assert.True(t, lgr.IsLevelEnabled(logr.Error).Enabled)
	logger.Error("after reset")
}

// TestCacheConcurrentAddRemove tests cache behavior when targets are
// added and removed concurrently with logging.
func TestCacheConcurrentAddRemove(t *testing.T) {
	lgr, err := logr.New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	formatter := &formatters.Plain{Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Error}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutine adding/removing targets
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			target := targets.NewWriterTarget(nil)
			err := lgr.AddTarget(target, "dynamic", filter, formatter, 100)
			assert.NoError(t, err)

			// Small chance to reset cache explicitly
			if i%5 == 0 {
				lgr.ResetLevelCache()
			}

			// Remove the target
			err = lgr.RemoveTargets(context.Background(), func(ti logr.TargetInfo) bool {
				return ti.Name == "dynamic"
			})
			assert.NoError(t, err)
		}
		// Close stopCh when add/remove is done to signal logging goroutines to stop
		close(stopCh)
	}()

	// Goroutines logging concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger := lgr.NewLogger()
			for {
				select {
				case <-stopCh:
					return
				default:
					logger.Error("concurrent log")
					lgr.IsLevelEnabled(logr.Error)
				}
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// If we got here without deadlock or panic, the test passes
}
