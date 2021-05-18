package formatters

import (
	"bytes"
	"fmt"

	"github.com/mattermost/logr/v2"
)

// Plain is the simplest formatter, outputting only text with
// no colors.
type Plain struct {
	// DisableTimestamp disables output of timestamp field.
	DisableTimestamp bool `json:"disable_timestamp"`
	// DisableLevel disables output of level field.
	DisableLevel bool `json:"disable_level"`
	// DisableMsg disables output of msg field.
	DisableMsg bool `json:"disable_msg"`
	// DisableFields disables output of all fields.
	DisableFields bool `json:"disable_fields"`
	// DisableStacktrace disables output of stack trace.
	DisableStacktrace bool `json:"disable_stack_trace"`

	// Delim is an optional delimiter output between each log field.
	// Defaults to a single space.
	Delim string `json:"delim"`

	// TimestampFormat is an optional format for timestamps. If empty
	// then DefTimestampFormat is used.
	TimestampFormat string `json:"timestamp_format"`
}

// Format converts a log record to bytes.
func (p *Plain) Format(rec *logr.LogRec, stacktrace bool, buf *bytes.Buffer) (*bytes.Buffer, error) {
	delim := p.Delim
	if delim == "" {
		delim = " "
	}
	if buf == nil {
		buf = &bytes.Buffer{}
	}

	timestampFmt := p.TimestampFormat
	if timestampFmt == "" {
		timestampFmt = logr.DefTimestampFormat
	}

	if !p.DisableTimestamp {
		var arr [128]byte
		tbuf := rec.Time().AppendFormat(arr[:0], timestampFmt)
		buf.Write(tbuf)
		buf.WriteString(delim)
	}
	if !p.DisableLevel {
		fmt.Fprintf(buf, "%v%s", rec.Level().Name, delim)
	}
	if !p.DisableMsg {
		fmt.Fprint(buf, rec.Msg(), delim)
	}
	if !p.DisableFields {
		fields := rec.Fields()
		if len(fields) > 0 {
			if err := logr.WriteFields(buf, fields, logr.Space); err != nil {
				return nil, err
			}
		}
	}
	if stacktrace && !p.DisableStacktrace {
		frames := rec.StackFrames()
		if len(frames) > 0 {
			buf.WriteString("\n")
			if err := logr.WriteStacktrace(buf, rec.StackFrames()); err != nil {
				return nil, err
			}
		}
	}
	buf.WriteString("\n")
	return buf, nil
}
