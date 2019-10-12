package logr

import (
	"fmt"
)

// LevelID is the unique id of each level.
type LevelID uint8

// Level provides a mechanism to enable/disable specific log lines.
// A default implementation using "panic, fatal..." is provided, and
// a more flexible alternative implementation is also provided that
// allows any number of custom levels.
type Level interface {
	fmt.Stringer
	ID() LevelID
}

// Filter allows targets to determine which Level(s) are active
// for logging and which Level(s) require a stack trace to be output.
type Filter interface {
	IsEnabled(Level) bool
	IsStacktraceEnabled(Level) bool
}
