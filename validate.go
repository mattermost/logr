package logr

import (
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

// TruncateWriter is where TruncateOptionText reports a truncation, instead of
// hardcoding where that goes. It defaults to os.Stderr; point it at your own
// io.Writer (a file, a pipe into your logger, etc.) to receive these instead.
var TruncateWriter io.Writer = os.Stderr

// MaxDelimLen is the maximum length, in bytes, of a field delimiter. A value longer than
// this is truncated rather than rejected; see TruncateOptionText.
const MaxDelimLen = 3

// MaxLineEndLen is the maximum length, in bytes, of a line terminator. A value longer
// than this is truncated rather than rejected; see TruncateOptionText.
//
// A line terminator is written after every log record, so unrestricted
// content there is enough to write an arbitrary payload to a target
// regardless of what is logged. Capping it this tightly rather than
// validating its content keeps the option too short to carry a payload
// without failing configuration that would otherwise still work.
const MaxLineEndLen = 3

// MaxLevelNameLen is the maximum length, in bytes, of a level name that comes from
// configuration. A longer name is truncated rather than rejected; see
// TruncateOptionText. It also caps min_level_len, since padding past the
// longest permitted name has no purpose.
//
// A level name is written at the start of every log record, so an unrestricted
// name lets configuration choose what the target file starts with.
const MaxLevelNameLen = 64

// MaxTimestampFormatLen is the maximum length, in bytes, of a timestamp_format. A longer
// value is truncated rather than rejected; see TruncateOptionText.
//
// Go layouts copy most characters literally into every record, so an
// unrestricted layout is another way to write a payload to the target.
const MaxTimestampFormatLen = 64

// TruncateOptionText truncates *s to at most maxLen bytes in place and reports
// it via TruncateWriter, instead of failing validation. The cut falls on a
// character boundary, so a multi-byte character is dropped whole rather than
// split. It is meant for options such as a field delimiter or line terminator:
// punctuation copied verbatim into every record, where an oversized value can
// be fixed up without losing anything the value was meant to convey, unlike a
// hostname or file path.
func TruncateOptionText(name string, s *string, maxLen int) {
	if len(*s) <= maxLen {
		return
	}
	if TruncateWriter != nil {
		fmt.Fprintf(TruncateWriter, "%s is too long (%d bytes, maximum %d), truncating\n", name, len(*s), maxLen)
	}
	cut := maxLen
	for cut > 0 && !utf8.RuneStart((*s)[cut]) {
		cut--
	}
	*s = (*s)[:cut]
}
