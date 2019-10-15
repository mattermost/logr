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

	// KeyTimestamp overrides the timestamp field key name.
	KeyTimestamp string

	// KeyLevel overrides the level field key name.
	KeyLevel string

	// KeyMsg overrides the msg field key name.
	KeyMsg string

	// KeyContextFields when not empty will group all context fields
	// under this key.
	KeyContextFields string

	// KeyStacktrace overrides the stacktrace field key name.
	KeyStacktrace string
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
func (j *JSON) Format(rec *logr.LogRec, stacktrace bool) ([]byte, error) {
	timestampFmt := j.TimestampFormat
	if timestampFmt == "" {
		timestampFmt = logr.DefTimestampFormat
	}

	data := make(map[string]interface{}, 7)
	j.applyDefaultKeyNames()

	if !j.DisableTimestamp {
		var arr [128]byte
		tbuf := rec.Time().AppendFormat(arr[:0], timestampFmt)
		data[j.KeyTimestamp] = string(tbuf)
	}
	if !j.DisableLevel {
		data[j.KeyLevel] = rec.Level().Name
	}
	if !j.DisableMsg {
		data[j.KeyMsg] = rec.Msg()
	}
	if !j.DisableContext {
		if j.KeyContextFields != "" {
			data[j.KeyContextFields] = rec.Fields()
		} else {
			m := rec.Fields()
			if len(m) > 0 {
				m = prefixCollisions(data, m)
				for k, v := range m {
					switch v := v.(type) {
					case error:
						data[k] = v.Error()
					default:
						data[k] = v
					}
				}
			}
		}
	}
	if stacktrace && !j.DisableStacktrace {
		frames := rec.StackFrames()
		numFrames := len(frames)
		if numFrames > 0 {
			st := make([]stacktraceRec, 0, numFrames)
			for _, frame := range frames {
				srec := stacktraceRec{
					Function: frame.Function,
					File:     frame.File,
					Line:     frame.Line,
				}
				st = append(st, srec)
			}
			data[j.KeyStacktrace] = st
		}
	}

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", j.Indent)
	encoder.SetEscapeHTML(j.EscapeHTML)

	err := encoder.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (j *JSON) applyDefaultKeyNames() {
	if j.KeyTimestamp == "" {
		j.KeyTimestamp = "timestamp"
	}
	if j.KeyLevel == "" {
		j.KeyLevel = "level"
	}
	if j.KeyMsg == "" {
		j.KeyMsg = "msg"
	}
	if j.KeyStacktrace == "" {
		j.KeyStacktrace = "stacktrace"
	}
}

func prefixCollisions(data map[string]interface{}, m map[string]interface{}) map[string]interface{} {
	// first check if there are any collisions to avoid creating a new map. This will be the
	// case most of the time.
	var collision bool
	for k := range m {
		if _, ok := data[k]; ok {
			collision = true
			break
		}
	}
	if !collision {
		return m
	}

	out := make(map[string]interface{}, len(data)+len(m))
	for k, v := range m {
		if _, ok := data[k]; ok {
			out["ctx."+k] = v
		} else {
			out[k] = v
		}
	}
	return out
}
