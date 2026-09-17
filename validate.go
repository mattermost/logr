package logr

import (
	"fmt"
	"os"
)

// MaxDelimLen is the maximum length of a field delimiter. A value longer than
// this is truncated rather than rejected; see TruncateOptionText.
const MaxDelimLen = 3

// MaxLineEndLen is the maximum length of a line terminator. A value longer
// than this is truncated rather than rejected; see TruncateOptionText.
//
// A line terminator is written after every log record, so unrestricted
// content there is enough to write an arbitrary payload to a target
// regardless of what is logged. Capping it this tightly rather than
// validating its content keeps the option too short to carry a payload
// without failing configuration that would otherwise still work.
const MaxLineEndLen = 3

// TruncateOptionText truncates *s to maxLen characters in place and logs to
// stderr, instead of failing validation. It is meant for options such as a
// field delimiter or line terminator: punctuation copied verbatim into every
// record, where an oversized value can be fixed up without losing anything
// the value was meant to convey, unlike a hostname or file path.
func TruncateOptionText(name string, s *string, maxLen int) {
	runes := []rune(*s)
	if len(runes) <= maxLen {
		return
	}
	fmt.Fprintf(os.Stderr, "%s is too long (%d characters, maximum %d), truncating\n", name, len(runes), maxLen)
	*s = string(runes[:maxLen])
}
