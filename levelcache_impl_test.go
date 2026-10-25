package logr

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAtomicLevelCache(t *testing.T) {
	c := &atomicLevelCache{}

	_, ok := c.get(Info.ID)
	require.False(t, ok)

	require.NoError(t, c.put(Info.ID, LevelStatus{Enabled: true, Stacktrace: true}))
	status, ok := c.get(Info.ID)
	require.True(t, ok)
	require.True(t, status.Enabled)
	require.True(t, status.Stacktrace)

	require.NoError(t, c.put(Warn.ID, LevelStatus{}))
	status, ok = c.get(Warn.ID)
	require.True(t, ok)
	require.False(t, status.Enabled)
	require.False(t, status.Stacktrace)

	require.NoError(t, c.put(MaxLevelID, LevelStatus{Enabled: true}))
	_, ok = c.get(MaxLevelID)
	require.True(t, ok)

	c.clear()
	for _, id := range []LevelID{Info.ID, Warn.ID, MaxLevelID} {
		_, ok = c.get(id)
		require.False(t, ok, "level id %d should be cleared", id)
	}

	require.Error(t, c.put(MaxLevelID+1, LevelStatus{Enabled: true}))
	_, ok = c.get(MaxLevelID + 1)
	require.False(t, ok)
}

func TestAtomicLevelCacheGenerationRollover(t *testing.T) {
	c := &atomicLevelCache{}
	c.generation.Store(atomicLevelMaxGeneration)

	require.NoError(t, c.put(Error.ID, LevelStatus{Enabled: true}))
	status, ok := c.get(Error.ID)
	require.True(t, ok)
	require.True(t, status.Enabled)

	c.clear()

	require.Equal(t, uint32(1), c.generation.Load())
	_, ok = c.get(Error.ID)
	require.False(t, ok)

	require.NoError(t, c.put(Warn.ID, LevelStatus{Enabled: true, Stacktrace: true}))
	status, ok = c.get(Warn.ID)
	require.True(t, ok)
	require.True(t, status.Stacktrace)

	_, ok = c.get(Error.ID)
	require.False(t, ok)
}

// TestAtomicLevelCacheRolloverDiscardsRacingPut verifies that a put racing with
// a generation rollover cannot leave an entry stamped with the pre-reset
// generation, which would read as valid again after the next wrap.
func TestAtomicLevelCacheRolloverDiscardsRacingPut(t *testing.T) {
	c := &atomicLevelCache{}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				if err := c.put(LevelID(i%256), LevelStatus{Enabled: true}); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	for i := 0; i < 200; i++ {
		c.generation.Store(atomicLevelMaxGeneration)
		c.clear()
	}

	close(stop)
	wg.Wait()

	for id := 0; id <= MaxLevelID; id++ {
		if c.arr[id].Load()>>atomicLevelGenerationShift == atomicLevelMaxGeneration {
			t.Fatalf("level id %d survived the rollover reset with the old generation", id)
		}
	}
}

// TestAtomicLevelCacheDiscardsStalePut replays the exact interleaving where a
// put's store lands after a rollover reset has zeroed the array and restarted
// generations, so the entry would otherwise read as valid at the next wrap.
func TestAtomicLevelCacheDiscardsStalePut(t *testing.T) {
	c := &atomicLevelCache{}
	c.generation.Store(atomicLevelMaxGeneration)

	require.NoError(t, c.put(Warn.ID, LevelStatus{Enabled: true}))
	stale := c.arr[Warn.ID].Load()

	c.clear()
	require.Equal(t, uint32(1), c.generation.Load())

	c.arr[Warn.ID].Store(stale)
	c.discardIfStale(Warn.ID, stale, atomicLevelMaxGeneration)

	c.generation.Store(atomicLevelMaxGeneration)
	_, ok := c.get(Warn.ID)
	require.False(t, ok, "entry from before the rollover became valid again")
}

// TestAtomicLevelCacheZeroValueClear covers clearing a cache before anything
// has been cached, where the generation still has to be initialized.
func TestAtomicLevelCacheZeroValueClear(t *testing.T) {
	c := &atomicLevelCache{}

	c.clear()
	require.Equal(t, uint32(1), c.generation.Load())

	require.NoError(t, c.put(Info.ID, LevelStatus{Enabled: true}))
	status, ok := c.get(Info.ID)
	require.True(t, ok)
	require.True(t, status.Enabled)
}

// TestAtomicLevelCacheConcurrentClearsAcrossRollover verifies that concurrent
// clears crossing the rollover each advance the generation exactly once.
func TestAtomicLevelCacheConcurrentClearsAcrossRollover(t *testing.T) {
	const (
		clearsBeforeMax = 50
		workers         = 8
		clearsPerWorker = 25
	)

	c := &atomicLevelCache{}
	c.generation.Store(atomicLevelMaxGeneration - clearsBeforeMax)
	require.NoError(t, c.put(Error.ID, LevelStatus{Enabled: true}))

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < clearsPerWorker; i++ {
				c.clear()
			}
		}()
	}
	wg.Wait()

	// The clear that reaches past the maximum lands on generation 1, so the
	// remaining clears count up from there.
	require.Equal(t, uint32(workers*clearsPerWorker-clearsBeforeMax), c.generation.Load())
	require.Zero(t, c.arr[Error.ID].Load(), "rollover did not zero the array")
	_, ok := c.get(Error.ID)
	require.False(t, ok)
}

// TestAtomicLevelCacheDuringRolloverReset verifies the cache behaves while
// another goroutine is in the middle of a rollover reset.
func TestAtomicLevelCacheDuringRolloverReset(t *testing.T) {
	t.Run("get misses", func(t *testing.T) {
		c := &atomicLevelCache{}
		c.arr[Error.ID].Store(atomicLevelMaxGeneration<<atomicLevelGenerationShift | atomicLevelEnabledBit)
		c.generation.Store(atomicLevelResetting)

		_, ok := c.get(Error.ID)
		require.False(t, ok)
	})

	t.Run("put waits for the new generation", func(t *testing.T) {
		c := &atomicLevelCache{}
		c.generation.Store(atomicLevelResetting)

		done := make(chan error, 1)
		go func() { done <- c.put(Warn.ID, LevelStatus{Enabled: true}) }()

		c.generation.Store(1)
		require.NoError(t, <-done)

		status, ok := c.get(Warn.ID)
		require.True(t, ok)
		require.True(t, status.Enabled)
	})

	t.Run("clear waits for the new generation", func(t *testing.T) {
		c := &atomicLevelCache{}
		c.generation.Store(atomicLevelResetting)

		done := make(chan struct{})
		go func() {
			c.clear()
			close(done)
		}()

		c.generation.Store(1)
		<-done
		require.Equal(t, uint32(2), c.generation.Load())
	})
}

// TestTargetLevelCacheKeepsStacktrace verifies a per-target cache hit reports
// the filter's stack trace setting, including when only the top-level cache is
// cold.
func TestTargetLevelCacheKeepsStacktrace(t *testing.T) {
	lgr, err := New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()

	filter := &StdFilter{Lvl: Error, Stacktrace: Error}
	err = lgr.AddTarget(discardTarget{}, "test", filter, &DefaultFormatter{}, 100)
	require.NoError(t, err)
	host := lgr.targetHosts[0]

	enabled, level := host.IsLevelEnabled(Error)
	require.True(t, enabled)
	require.True(t, level.Stacktrace, "cache miss")

	enabled, level = host.IsLevelEnabled(Error)
	require.True(t, enabled)
	require.True(t, level.Stacktrace, "cache hit")

	lgr.lvlCache.clear()
	require.True(t, lgr.IsLevelEnabled(Error).Stacktrace)
}

// gatedFilter blocks its first GetEnabledLevel call until released, after it
// has read the enabled setting.
type gatedFilter struct {
	enabled atomic.Bool
	gate    atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (f *gatedFilter) GetEnabledLevel(level Level) (Level, bool) {
	enabled := f.enabled.Load()
	if f.gate.CompareAndSwap(true, false) {
		close(f.entered)
		<-f.release
	}
	return level, enabled
}

// TestResetLevelCacheDuringCacheMiss verifies that a cache miss which read the
// filter before a ResetLevelCache cannot leave its result cached afterwards.
func TestResetLevelCacheDuringCacheMiss(t *testing.T) {
	filter := &gatedFilter{entered: make(chan struct{}), release: make(chan struct{})}
	filter.enabled.Store(true)
	filter.gate.Store(true)

	lgr, err := New()
	require.NoError(t, err)
	defer func() { require.NoError(t, lgr.Shutdown()) }()
	require.NoError(t, lgr.AddTarget(discardTarget{}, "test", filter, &DefaultFormatter{}, 100))

	lookup := make(chan LevelStatus, 1)
	go func() { lookup <- lgr.IsLevelEnabled(Error) }()
	<-filter.entered

	filter.enabled.Store(false)
	resetDone := make(chan struct{})
	go func() {
		lgr.ResetLevelCache()
		close(resetDone)
	}()

	// Give an unsynchronized reset time to finish before the lookup stores its result.
	select {
	case <-resetDone:
	case <-time.After(50 * time.Millisecond):
	}
	close(filter.release)

	require.True(t, (<-lookup).Enabled)
	<-resetDone
	require.False(t, lgr.IsLevelEnabled(Error).Enabled, "stale entry survived ResetLevelCache")
	enabled, _ := lgr.targetHosts[0].IsLevelEnabled(Error)
	require.False(t, enabled, "stale per-target entry survived ResetLevelCache")
}
