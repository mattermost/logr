package level

import (
	"fmt"
)

// Level provides a mechanism to enable/disable specific log lines.
// A default implementation using "panic, fatal..." is provided, however
// more flexible implementations are possible such as topic names.
type Level interface {
	fmt.Stringer
	IsEnabled(Level) bool
	IsStacktraceEnabled(Level) bool
}
