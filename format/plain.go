package format

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/wiggin77/logr"
)

const (
	// DefTimestampFormat is the default time stamp format used by
	// Plain formatter and others.
	DefTimestampFormat = "2006-01-02 15:04:05.000 Z07:00"
)

// Plain is the simplest formatter, outputting only text with
// no colors.
type Plain struct {
	// DisableTimestamp disables output of timestamp field.
	DisableTimestamp bool
	// DisableLevel disables output of level field.
	DisableLevel bool
	// DisableMsg disables output of msg field.
	DisableMsg bool
	// DisableContext disables output of all context fields.
	DisableContext bool
	// DisableStacktrace disables output of stack trace.
	DisableStacktrace bool

	// Delim is an optional delimiter output between each log field.
	// Defaults to a single space.
	Delim string

	// TimestampFormat is an optional format for timestamps. If empty
	// then DefTimestampFormat is used.
	TimestampFormat string
}

// Format converts a log record to bytes.
func (p *Plain) Format(rec *logr.LogRec) ([]byte, error) {
	sb := &strings.Builder{}

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
	if !p.DisableContext {
		fmt.Fprint(sb, "ctx:{")
		writeFieldsPlain(sb, rec.Fields(), ", ")
		fmt.Fprint(sb, "}")
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
