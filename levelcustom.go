package logr

import (
	"sync"
)

// CustomFilter allows targets to enable logging via a list of levels.
type CustomFilter struct {
	mux    sync.RWMutex
	levels map[LevelID]Level
}

// GetEnabledLevel returns the Level with the specified Level.ID and whether the level
// is enabled for this filter.
func (st *CustomFilter) GetEnabledLevel(level Level) (Level, bool) {
	st.mux.RLock()
	defer st.mux.RUnlock()
	levelEnabled, ok := st.levels[level.ID]

	if ok && levelEnabled.Name == "" {
		levelEnabled.Name = level.Name
	}

	return levelEnabled, ok
}

// Add adds one or more levels to the list. Adding a level enables logging for
// that level on any targets using this CustomFilter.
func (st *CustomFilter) Add(levels ...Level) {
	st.mux.Lock()
	defer st.mux.Unlock()

	if st.levels == nil {
		st.levels = make(map[LevelID]Level)
	}

	for _, s := range levels {
		st.levels[s.ID] = s
	}
}
