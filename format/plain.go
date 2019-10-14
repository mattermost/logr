package format

import (
	"fmt"
	"strings"

	"github.com/wiggin77/logr"
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
func (p *Plain) Format(rec *logr.LogRec, stacktrace bool) ([]byte, error) {
	sb := &strings.Builder{}

	delim := p.Delim
	if delim == "" {
		delim = " "
	}

	timestampFmt := p.TimestampFormat
	if timestampFmt == "" {
		timestampFmt = logr.DefTimestampFormat
	}

	if !p.DisableTimestamp {
		var arr [128]byte
		tbuf := rec.Time().AppendFormat(arr[:0], timestampFmt)
		sb.Write(tbuf)
		sb.WriteString(delim)
	}
	if !p.DisableLevel {
		fmt.Fprintf(sb, "%v%s", rec.Level().Name, delim)
	}
	if !p.DisableMsg {
		fmt.Fprint(sb, rec.Msg(), delim)
	}
	if !p.DisableContext {
		ctx := rec.Fields()
		if len(ctx) > 0 {
			logr.WriteFields(sb, ctx, " ")
		}
	}
	if stacktrace && !p.DisableStacktrace {
		frames := rec.StackFrames()
		if len(frames) > 0 {
			sb.WriteString("\n")
			logr.WriteStacktrace(sb, rec.StackFrames())
		}
	}
	sb.WriteString("\n")
	return []byte(sb.String()), nil
}
