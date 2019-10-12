package logr

import "sync"

// LevelStatus represents whether a level is enabled and
// requires a stack trace.
type LevelStatus struct {
	Enabled    bool
	Stacktrace bool
}

type levelCache interface {
	get(id int) (LevelStatus, bool)
	put(id int, status LevelStatus)
	clear()
}

// syncMapLevelCache uses sync.Map which may better handle large concurrency
// scenarios.
type syncMapLevelCache struct {
	m sync.Map
}

func (c *syncMapLevelCache) get(id int) (LevelStatus, bool) {
	status, ok := c.m.Load(id)
	return status.(LevelStatus), ok
}

func (c *syncMapLevelCache) put(id int, status LevelStatus) {
	c.m.Store(id, status)
}

func (c *syncMapLevelCache) clear() {
	c.m.Range(func(key interface{}, value interface{}) bool {
		c.m.Delete(key)
		return true
	})
}

// mapLevelCache uses map and a mutex.
type mapLevelCache struct {
	m   map[int]LevelStatus
	mux sync.RWMutex
}

func (c *mapLevelCache) get(id int) (LevelStatus, bool) {
	c.mux.RLock()
	status, ok := c.m[id]
	c.mux.RUnlock()
	return status, ok
}

func (c *mapLevelCache) put(id int, status LevelStatus) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.m[id] = status
}

func (c *mapLevelCache) clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	size := len(c.m)
	c.m = make(map[int]LevelStatus, size)
}
