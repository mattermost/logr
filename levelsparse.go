package logr

import (
	"sync"
)

// SparseLevel can be used as part of a list of Level where
// any levels in the list are enabled.
type SparseLevel struct {
	FID        int
	Name       string
	Stacktrace bool
}

// ID returns the unique id of this sparse filter.
func (s SparseLevel) ID() int {
	return s.FID
}

// String returns a string representation of this sparse filter.
func (s SparseLevel) String() string {
	return s.Name
}

// SparseFilter allows targets to enable logging via a list of levels.
type SparseFilter struct {
	mux    sync.RWMutex
	levels map[int]SparseLevel
}

// IsEnabled returns true if the specifed Level exists in this list.
func (st *SparseFilter) IsEnabled(level Level) bool {
	lvl, ok := level.(SparseLevel)
	if !ok {
		return false
	}
	st.mux.RLock()
	defer st.mux.RUnlock()
	_, ok = st.levels[lvl.FID]
	return ok
}

// IsStacktraceEnabled returns true if the specifed Level requires a stack trace.
func (st *SparseFilter) IsStacktraceEnabled(level Level) bool {
	lvl, ok := level.(*SparseLevel)
	if !ok {
		return false
	}
	st.mux.RLock()
	defer st.mux.RUnlock()
	sl, ok := st.levels[lvl.FID]
	if ok {
		return sl.Stacktrace
	}
	return false
}

// Add adds one or more levels to the list. Adding a level
// enables logging for that level on any targets using this list.
func (st *SparseFilter) Add(levels []SparseLevel) {
	st.mux.Lock()
	defer st.mux.Unlock()
	for _, s := range levels {
		st.levels[s.FID] = s
	}
}
