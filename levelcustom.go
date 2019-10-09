package logr

import (
	"sync"
)

// CustomLevel can be used as part of a list of Levels where
// any levels in the list are enabled.
type CustomLevel struct {
	FID        int
	Name       string
	Stacktrace bool
}

// ID returns the unique id of this custom level.
func (s CustomLevel) ID() int {
	return s.FID
}

// String returns a string representation of this custom level.
func (s CustomLevel) String() string {
	return s.Name
}

// CustomFilter allows targets to enable logging via a list of levels.
type CustomFilter struct {
	mux    sync.RWMutex
	levels map[int]Level
}

// IsEnabled returns true if the specified Level exists in this list.
func (st *CustomFilter) IsEnabled(level Level) bool {
	st.mux.RLock()
	defer st.mux.RUnlock()
	_, ok := st.levels[level.ID()]
	return ok
}

// IsStacktraceEnabled returns true if the specified Level requires a stack trace.
func (st *CustomFilter) IsStacktraceEnabled(level Level) bool {
	st.mux.RLock()
	defer st.mux.RUnlock()
	lvl, ok := st.levels[level.ID()]
	if ok {
		cl, ok := lvl.(CustomLevel)
		if ok {
			return cl.Stacktrace
		}
	}
	return false
}

// Add adds one or more levels to the list. Adding a level enables logging for
// that level on any targets using this CustomFilter.
func (st *CustomFilter) Add(levels ...Level) {
	st.mux.Lock()
	defer st.mux.Unlock()

	if st.levels == nil {
		st.levels = make(map[int]Level)
	}

	for _, s := range levels {
		st.levels[s.ID()] = s
	}
}
