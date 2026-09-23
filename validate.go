package logr

import (
	"fmt"
	"io"
	"os"
)

// TruncateWriter is where TruncateOptionText reports a truncation, instead of
// hardcoding where that goes. It defaults to os.Stderr; point it at your own
// io.Writer (a file, a pipe into your logger, etc.) to receive these instead.
var TruncateWriter io.Writer = os.Stderr

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

// TruncateOptionText truncates *s to maxLen characters in place and reports
// it via TruncateWriter, instead of failing validation. It is meant for
// options such as a field delimiter or line terminator: punctuation copied
// verbatim into every record, where an oversized value can be fixed up
// without losing anything the value was meant to convey, unlike a hostname or
// file path.
func TruncateOptionText(name string, s *string, maxLen int) {
	runes := []rune(*s)
	if len(runes) <= maxLen {
		return
	}
	if TruncateWriter != nil {
		fmt.Fprintf(TruncateWriter, "%s is too long (%d characters, maximum %d), truncating\n", name, len(runes), maxLen)
	}
	*s = string(runes[:maxLen])
}
