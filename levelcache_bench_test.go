package logr

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// spreadLevels is the number of distinct level ids used by the "spread"
// benchmarks. Must be a power of two.
const spreadLevels = 256

// benchSink prevents the compiler from eliminating benchmark work.
var benchSink atomic.Uint64

func newBenchLevelCache(b *testing.B) *atomicLevelCache {
	c := &atomicLevelCache{}
	c.setup()
	b.ReportAllocs()
	return c
}

func primeLevelCache(b *testing.B, c *atomicLevelCache, id LevelID) {
	if err := c.put(id, LevelStatus{Enabled: true}); err != nil {
		b.Fatal(err)
	}
	if _, ok := c.get(id); !ok {
		b.Fatalf("cache miss after put for level id %d", id)
	}
}

func BenchmarkLevelCacheGetHit(b *testing.B) {
	c := newBenchLevelCache(b)
	primeLevelCache(b, c, Error.ID)

	var acc uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if status, ok := c.get(Error.ID); ok && status.Enabled {
			acc++
		}
	}
	b.StopTimer()
	benchSink.Add(acc)
}

func BenchmarkLevelCacheGetMiss(b *testing.B) {
	c := newBenchLevelCache(b)

	const id LevelID = 5000
	if _, ok := c.get(id); ok {
		b.Fatalf("cache hit for level id %d that was never put", id)
	}

	var acc uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := c.get(id); !ok {
			acc++
		}
	}
	b.StopTimer()
	benchSink.Add(acc)
}

func BenchmarkLevelCacheGetSpread(b *testing.B) {
	c := newBenchLevelCache(b)
	for i := 0; i < spreadLevels; i++ {
		primeLevelCache(b, c, LevelID(i))
	}

	var acc uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if status, ok := c.get(LevelID(i & (spreadLevels - 1))); ok && status.Enabled {
			acc++
		}
	}
	b.StopTimer()
	benchSink.Add(acc)
}

func BenchmarkLevelCachePut(b *testing.B) {
	c := newBenchLevelCache(b)
	status := LevelStatus{Enabled: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.put(LevelID(i&(spreadLevels-1)), status); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLevelCacheClear(b *testing.B) {
	c := newBenchLevelCache(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.clear()
	}
}

func BenchmarkLevelCacheGetHitParallel(b *testing.B) {
	c := newBenchLevelCache(b)
	primeLevelCache(b, c, Error.ID)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var acc uint64
		for pb.Next() {
			if status, ok := c.get(Error.ID); ok && status.Enabled {
				acc++
			}
		}
		benchSink.Add(acc)
	})
}

func BenchmarkLevelCacheGetSpreadParallel(b *testing.B) {
	c := newBenchLevelCache(b)
	for i := 0; i < spreadLevels; i++ {
		primeLevelCache(b, c, LevelID(i))
	}

	var offset atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := int(offset.Add(1)) * 7
		var acc uint64
		for pb.Next() {
			if status, ok := c.get(LevelID(i & (spreadLevels - 1))); ok && status.Enabled {
				acc++
			}
			i++
		}
		benchSink.Add(acc)
	})
}

// BenchmarkLevelCacheReadWriteParallel mixes one put per 100 gets, simulating
// cache refills after a reset.
func BenchmarkLevelCacheReadWriteParallel(b *testing.B) {
	c := newBenchLevelCache(b)
	primeLevelCache(b, c, Error.ID)
	status := LevelStatus{Enabled: true}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var acc uint64
		i := 0
		for pb.Next() {
			i++
			if i%100 == 0 {
				if err := c.put(Error.ID, status); err != nil {
					b.Error(err)
					return
				}
				continue
			}
			if s, ok := c.get(Error.ID); ok && s.Enabled {
				acc++
			}
		}
		benchSink.Add(acc)
	})
}

// BenchmarkLevelCacheGetWithClears measures readers while a writer invalidates
// the cache, the pattern produced by AddTarget/SetOption/ResetLevelCache.
func BenchmarkLevelCacheGetWithClears(b *testing.B) {
	c := newBenchLevelCache(b)
	primeLevelCache(b, c, Error.ID)

	var stop atomic.Bool
	done := make(chan struct{})
	go func() {
		defer close(done)
		for !stop.Load() {
			c.clear()
			_ = c.put(Error.ID, LevelStatus{Enabled: true})
			time.Sleep(50 * time.Microsecond)
		}
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var acc uint64
		for pb.Next() {
			if status, ok := c.get(Error.ID); ok && status.Enabled {
				acc++
			}
		}
		benchSink.Add(acc)
	})
	b.StopTimer()

	stop.Store(true)
	<-done
}

type discardTarget struct{}

func (discardTarget) Init() error                            { return nil }
func (discardTarget) Write(p []byte, _ *LogRec) (int, error) { return len(p), nil }
func (discardTarget) Shutdown() error                        { return nil }

func newBenchLogr(b *testing.B) *Logr {
	lgr, err := New()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		err = lgr.AddTarget(discardTarget{}, "bench"+strconv.Itoa(i),
			&StdFilter{Lvl: Error}, &DefaultFormatter{}, 1000)
		if err != nil {
			b.Fatal(err)
		}
	}

	if !lgr.IsLevelEnabled(Error).Enabled {
		b.Fatal("Error level should be enabled")
	}

	b.ReportAllocs()
	return lgr
}

func BenchmarkIsLevelEnabledCached(b *testing.B) {
	lgr := newBenchLogr(b)

	var acc uint64
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if lgr.IsLevelEnabled(Error).Enabled {
			acc++
		}
	}
	b.StopTimer()
	benchSink.Add(acc)

	if err := lgr.Shutdown(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkIsLevelEnabledCachedParallel(b *testing.B) {
	lgr := newBenchLogr(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var acc uint64
		for pb.Next() {
			if lgr.IsLevelEnabled(Error).Enabled {
				acc++
			}
		}
		benchSink.Add(acc)
	})
	b.StopTimer()

	if err := lgr.Shutdown(); err != nil {
		b.Fatal(err)
	}
}
