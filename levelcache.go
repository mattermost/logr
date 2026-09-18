package logr

import (
	"fmt"
	"sync/atomic"
)

// LevelStatus represents whether a level is enabled and
// requires a stack trace.
type LevelStatus struct {
	Enabled    bool
	Stacktrace bool
}

type levelCache interface {
	setup()
	get(id LevelID) (LevelStatus, bool)
	put(id LevelID, status LevelStatus) error
	clear()
}

const (
	atomicLevelEnabledBit uint32 = 1 << iota
	atomicLevelStacktraceBit

	atomicLevelGenerationShift = 2

	// Two bits are used for status, leaving 30 bits for generation.
	atomicLevelMaxGeneration uint32 = (1 << (32 - atomicLevelGenerationShift)) - 1

	// Generation value used while performing the extremely rare rollover reset.
	atomicLevelResetting uint32 = ^uint32(0)
)

// atomicLevelCache is a lock-free level cache.
//
// Each entry contains:
//
//	31                              2 1 0
//	+--------------------------------+-+-+
//	|          generation            |S|E|
//	+--------------------------------+-+-+
//
// E = Enabled
// S = Stacktrace
//
// A generation mismatch means that the entry is empty/stale.
//
// The normal clear path is O(1): increment the generation.
// No entries need to be touched.
type atomicLevelCache struct {
	arr        [MaxLevelID + 1]atomic.Uint32
	generation atomic.Uint32
}

func (c *atomicLevelCache) setup() {
	// Generation zero represents an uninitialized cache.
	c.generation.CompareAndSwap(0, 1)
}

func (c *atomicLevelCache) get(id LevelID) (LevelStatus, bool) {
	if id > MaxLevelID {
		return LevelStatus{}, false
	}

	generation := c.generation.Load()
	if generation == 0 || generation == atomicLevelResetting {
		return LevelStatus{}, false
	}

	value := c.arr[id].Load()

	if value>>atomicLevelGenerationShift != generation {
		return LevelStatus{}, false
	}

	return LevelStatus{
		Enabled:    value&atomicLevelEnabledBit != 0,
		Stacktrace: value&atomicLevelStacktraceBit != 0,
	}, true
}

func (c *atomicLevelCache) put(id LevelID, status LevelStatus) error {
	if id > MaxLevelID {
		return fmt.Errorf("level id cannot exceed MaxLevelID (%d)", MaxLevelID)
	}

	var generation uint32

	for {
		generation = c.generation.Load()

		switch generation {
		case 0:
			// Allow the zero value to work even if setup() was not called.
			if c.generation.CompareAndSwap(0, 1) {
				generation = 1
			} else {
				continue
			}

		case atomicLevelResetting:
			continue
		}

		break
	}

	value := generation << atomicLevelGenerationShift

	if status.Enabled {
		value |= atomicLevelEnabledBit
	}

	if status.Stacktrace {
		value |= atomicLevelStacktraceBit
	}

	c.arr[id].Store(value)

	return nil
}

func (c *atomicLevelCache) clear() {
	for {
		generation := c.generation.Load()

		switch {
		case generation == 0:
			if c.generation.CompareAndSwap(0, 1) {
				return
			}

		case generation == atomicLevelResetting:
			continue

		case generation < atomicLevelMaxGeneration:
			if c.generation.CompareAndSwap(generation, generation+1) {
				return
			}

		default:
			if !c.generation.CompareAndSwap(
				atomicLevelMaxGeneration,
				atomicLevelResetting,
			) {
				continue
			}

			// Generation rollover is reached only after ~1 billion clears.
			// Physically clear the array so no old generation can become
			// valid when generations restart at 1.
			for i := range c.arr {
				c.arr[i].Store(0)
			}

			c.generation.Store(1)
			return
		}
	}
}
