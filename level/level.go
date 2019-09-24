package level

import (
	"fmt"
)

// Level provides a mechanism to enable/disable specific log lines.
// A default implementation using "panic, fatal..." is provided, and
// a more flexible alternative implementation is also provided that
// allows any number of custom levels.
type Level uint32

// TargetLevel determines which Level(s) are active for logging and
// which Level(s) require a stack trace to be output.
type TargetLevel interface {
	fmt.Stringer
	IsEnabled(Level) bool
	IsStacktraceEnabled(Level) bool
}
