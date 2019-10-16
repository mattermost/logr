package logr

import (
	"sync"
	"sync/atomic"
)

// LevelStatus represents whether a level is enabled and
// requires a stack trace.
type LevelStatus struct {
	Enabled    bool
	Stacktrace bool
	empty      bool
}

type levelCache interface {
	setup()
	get(id LevelID) (LevelStatus, bool)
	put(id LevelID, status LevelStatus)
	clear()
}

// syncMapLevelCache uses sync.Map which may better handle large concurrency
// scenarios.
type syncMapLevelCache struct {
	m sync.Map
}

func (c *syncMapLevelCache) setup() {
	c.clear()
}

func (c *syncMapLevelCache) get(id LevelID) (LevelStatus, bool) {
	s, _ := c.m.Load(id)
	status := s.(LevelStatus)
	return status, !status.empty
}

func (c *syncMapLevelCache) put(id LevelID, status LevelStatus) {
	c.m.Store(id, status)
}

func (c *syncMapLevelCache) clear() {
	var i LevelID
	for i = 0; i < 255; i++ {
		c.m.Store(i, LevelStatus{empty: true})
	}
}

// mapLevelCache using map and a mutex.
type mapLevelCache struct {
	m   map[LevelID]LevelStatus
	mux sync.RWMutex
}

func (c *mapLevelCache) setup() {
	c.m = make(map[LevelID]LevelStatus)
}

func (c *mapLevelCache) get(id LevelID) (LevelStatus, bool) {
	c.mux.RLock()
	status, ok := c.m[id]
	c.mux.RUnlock()
	return status, ok
}

func (c *mapLevelCache) put(id LevelID, status LevelStatus) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.m[id] = status
}

func (c *mapLevelCache) clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	size := len(c.m)
	c.m = make(map[LevelID]LevelStatus, size)
}

// arrayLevelCache using array and a mutex.
type arrayLevelCache struct {
	arr [256]LevelStatus
	mux sync.RWMutex
}

func (c *arrayLevelCache) setup() {
	c.clear()
}

var dummy = LevelStatus{}

func (c *arrayLevelCache) get(id LevelID) (LevelStatus, bool) {
	c.mux.RLock()
	status := c.arr[id]
	ok := !status.empty
	c.mux.RUnlock()
	return status, ok
}

func (c *arrayLevelCache) put(id LevelID, status LevelStatus) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.arr[id] = status
}

func (c *arrayLevelCache) clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	for i := range c.arr {
		c.arr[i] = LevelStatus{empty: true}
	}
}

// cowLevelCache using atomic value and map
type cowLevelCache struct {
	m   atomic.Value
	mux sync.Mutex
}

func (c *cowLevelCache) setup() {
	c.m.Store(make(map[LevelID]LevelStatus))
}

func (c *cowLevelCache) get(id LevelID) (LevelStatus, bool) {
	m1 := c.m.Load().(map[LevelID]LevelStatus)
	status, ok := m1[id]
	return status, ok
}

func (c *cowLevelCache) put(id LevelID, status LevelStatus) {
	c.mux.Lock()
	defer c.mux.Unlock()

	m1 := c.m.Load().(map[LevelID]LevelStatus)
	m2 := make(map[LevelID]LevelStatus, len(m1))
	for k, v := range m1 {
		m2[k] = v
	}
	m2[id] = status
	c.m.Store(m2)
}

func (c *cowLevelCache) clear() {
	c.mux.Lock()
	defer c.mux.Unlock()

	m1 := make(map[LevelID]LevelStatus)
	c.m.Store(m1)
}
