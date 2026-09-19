package logr

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLevelCacheAlwaysAtomic verifies the atomic cache is used for the Logr and
// per-target caches, including when the deprecated cache options are supplied.
func TestLevelCacheAlwaysAtomic(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
	}{
		{name: "default"},
		{name: "array option true", opts: []Option{UseArrayLevelCache(true)}},
		{name: "array option false", opts: []Option{UseArrayLevelCache(false)}},
		{name: "syncMap option true", opts: []Option{UseSyncMapLevelCache(true)}},
		{name: "syncMap option false", opts: []Option{UseSyncMapLevelCache(false)}},
	}

	const want = "*logr.atomicLevelCache"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lgr, err := New(tc.opts...)
			require.NoError(t, err)
			defer func() { require.NoError(t, lgr.Shutdown()) }()

			require.Equal(t, want, fmt.Sprintf("%T", lgr.lvlCache))

			err = lgr.AddTarget(discardTarget{}, "test", &StdFilter{Lvl: Error}, &DefaultFormatter{}, 100)
			require.NoError(t, err)
			require.Equal(t, want, fmt.Sprintf("%T", lgr.targetHosts[0].lvlCache))

			require.True(t, lgr.IsLevelEnabled(Error).Enabled)
			require.False(t, lgr.IsLevelEnabled(Debug).Enabled)
		})
	}
}

func TestAtomicLevelCache(t *testing.T) {
	c := &atomicLevelCache{}
	c.setup()

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

// TestAtomicLevelCacheZeroValue verifies the cache works without setup().
func TestAtomicLevelCacheZeroValue(t *testing.T) {
	c := &atomicLevelCache{}

	_, ok := c.get(Error.ID)
	require.False(t, ok)

	require.NoError(t, c.put(Error.ID, LevelStatus{Enabled: true}))
	status, ok := c.get(Error.ID)
	require.True(t, ok)
	require.True(t, status.Enabled)
}

func TestAtomicLevelCacheGenerationRollover(t *testing.T) {
	c := &atomicLevelCache{}
	c.setup()
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
	c.setup()

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
	c.setup()
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

// TestAtomicLevelCacheZeroValueClear covers clearing a cache that never had
// setup() called, where the generation still has to be initialized.
func TestAtomicLevelCacheZeroValueClear(t *testing.T) {
	c := &atomicLevelCache{}

	c.clear()
	require.Equal(t, uint32(1), c.generation.Load())

	require.NoError(t, c.put(Info.ID, LevelStatus{Enabled: true}))
	status, ok := c.get(Info.ID)
	require.True(t, ok)
	require.True(t, status.Enabled)
}
