package format

import (
	"bytes"
	"encoding/json"

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
	// DisableContext disables output of all context fields.
	DisableContext bool
	// DisableStacktrace disables output of stack trace.
	DisableStacktrace bool

	// TimestampFormat is an optional format for timestamps. If empty
	// then DefTimestampFormat is used.
	TimestampFormat string

	// Indent sets the character used to indent or pretty print the JSON.
	// Empty string means no pretty print.
	Indent string

	// EscapeHTML determines if certain characters (e.g. `<`, `>`, `&`)
	// are escaped.
	EscapeHTML bool
}

type jsonRec struct {
	Timestamp  string          `json:"timestamp,omitempty"`
	Level      string          `json:"level,omitempty"`
	Msg        string          `json:"msg,omitempty"`
	Ctx        logr.Fields     `json:"ctx,omitempty"`
	Stacktrace []stacktraceRec `json:"stacktrace,omitempty"`
}

type stacktraceRec struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Format converts a log record to bytes in JSON format.
func (j *JSON) Format(rec *logr.LogRec) ([]byte, error) {
	timestampFmt := j.TimestampFormat
	if timestampFmt == "" {
		timestampFmt = DefTimestampFormat
	}

	jrec := &jsonRec{}

	if !j.DisableTimestamp {
		jrec.Timestamp = rec.Time().Format(timestampFmt)
	}
	if !j.DisableLevel {
		jrec.Level = rec.Level().String()
	}
	if !j.DisableMsg {
		jrec.Msg = rec.Msg()
	}
	if !j.DisableContext {
		jrec.Ctx = rec.Fields()
	}
	if !j.DisableStacktrace {
		frames := rec.StackFrames()
		if len(frames) > 0 {
			for _, frame := range frames {
				srec := stacktraceRec{
					Function: frame.Function,
					File:     frame.File,
					Line:     frame.Line,
				}
				jrec.Stacktrace = append(jrec.Stacktrace, srec)
			}
		}
	}

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", j.Indent)
	encoder.SetEscapeHTML(j.EscapeHTML)

	err := encoder.Encode(jrec)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
