package format

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/wiggin77/logr"
)

// JSON formats log records as JSON.
type JSON struct {
	// DisableTimestamp disables output of timestamp field.
	DisableTimestamp bool
	// DisableLevel disables output of level field.
	DisableLevel bool
	// DisableMsg disables output of msg field.
	DisableMsg bool
	// DisableFields disables output of all context fields.
	DisableFields bool
	// DisableStacktrace disables output of stack trace.
	DisableStacktrace bool

	// TimestampFormat is an optional format for timestamps. If empty
	// then DefTimestampFormat is used.
	TimestampFormat string
}

// Format converts a log record to bytes in JSON format.
func (j *JSON) Format(rec *logr.LogRec) ([]byte, error) {
	buf := &bytes.Buffer

	delim := p.Delim
	if delim == "" {
		delim = " "
	}

	timestampFmt := p.TimestampFormat
	if timestampFmt == "" {
		timestampFmt = DefTimestampFormat
	}

	if !p.DisableTimestamp {
		fmt.Fprintf(sb, "%s%s", rec.Time().Format(timestampFmt), delim)
	}
	if !p.DisableLevel {
		fmt.Fprintf(sb, "%v%s", rec.Level(), delim)
	}
	if !p.DisableMsg {
		fmt.Fprintf(sb, "%s%s", rec.Msg(), delim)
	}
	if !p.DisableFields {
		writeFieldsPlain(sb, rec.Fields(), ", ")
	}
	if !p.DisableStacktrace {
		frames := rec.StackFrames()
		if len(frames) > 0 {
			sb.WriteString("\n")
			writeStacktracePlain(sb, rec.StackFrames())
		}
	}
	sb.WriteString("\n")

	return []byte(sb.String()), nil
}

// writeFieldsPlain writes zero or more name value pairs to the io.Writer.
// The pairs are sorted by key name and output in key=value format
// with optional separator between fields.
func writeFieldsPlain(w io.Writer, flds logr.Fields, separator string) {
	keys := make([]string, 0, len(flds))
	for k := range flds {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sep := ""
	for _, k := range keys {
		fmt.Fprintf(w, "%s%s=%v", sep, k, flds[k])
		sep = separator
	}
}

// writeStacktracePlain formats and outputs a stack trace to an io.Writer.
func writeStacktracePlain(w io.Writer, frames []runtime.Frame) {
	for _, frame := range frames {
		if frame.Function != "" {
			fmt.Fprintf(w, "  %s\n", frame.Function)
		}
		if frame.File != "" {
			fmt.Fprintf(w, "      %s:%d\n", frame.File, frame.Line)
		}
	}
}
