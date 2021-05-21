package formatters

import (
	"bytes"
	"fmt"
	"strings"

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
	DisableStacktrace bool `json:"disable_stacktrace"`

	// Delim is an optional delimiter output between each log field.
	// Defaults to a single space.
	Delim string `json:"delim"`

	// MinMessageLen sets the minimum msg length. If the msg text is less
	// than the minimum it will be padded with spaces.
	MinMessageLen int `json:"min_msg_len"`

	// TimestampFormat is an optional format for timestamps. If empty
	// then DefTimestampFormat is used.
	TimestampFormat string `json:"timestamp_format"`

	// LineEnd sets the end of line character(s). Defaults to '\n'.
	LineEnd string `json:"line_end"`

	// Color sets whether output should include color.
	Color bool `json:"color"`
}

func (p *Plain) CheckValid() error {
	if p.MinMessageLen < 0 || p.MinMessageLen > 1024 {
		return fmt.Errorf("min_msg_len is invaid(%d)", p.MinMessageLen)
	}
	return nil
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

	color := logr.NoColor
	if p.Color {
		color = rec.Level().Color
	}

	if !p.DisableLevel {
		_ = logr.WriteWithColor(buf, rec.Level().Name, color)
		buf.WriteString(delim)
	}

	if !p.DisableTimestamp {
		var arr [128]byte
		tbuf := rec.Time().AppendFormat(arr[:0], timestampFmt)
		buf.WriteByte('[')
		buf.Write(tbuf)
		buf.WriteByte(']')
		buf.WriteString(delim)
	}

	if !p.DisableMsg {
		count, _ := buf.WriteString(rec.Msg())
		if p.MinMessageLen > count {
			_, _ = buf.WriteString(strings.Repeat(" ", p.MinMessageLen-count))
		}
		_, _ = buf.WriteString(delim)
	}

	if !p.DisableFields {
		fields := rec.Fields()
		if len(fields) > 0 {
			if err := logr.WriteFields(buf, fields, logr.Space, color); err != nil {
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

	if p.LineEnd == "" {
		buf.WriteString("\n")
	} else {
		buf.WriteString(p.LineEnd)
	}

	return buf, nil
}
