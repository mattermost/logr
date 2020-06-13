package format

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"sort"

	"github.com/wiggin77/logr"
	"github.com/wiggin77/merror"
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
	if !p.DisableContext {
		ctx := rec.Fields()
		if len(ctx) > 0 {
			WriteFields(buf, ctx, " ")
		}
	}
	if stacktrace && !p.DisableStacktrace {
		frames := rec.StackFrames()
		if len(frames) > 0 {
			buf.WriteString("\n")
			WriteStacktrace(buf, rec.StackFrames())
		}
	}
	buf.WriteString("\n")
	return buf, nil
}

// WriteFields writes zero or more name value pairs to the io.Writer.
// The pairs are sorted by key name and output in key=value format
// with optional separator between fields.
func WriteFields(w io.Writer, flds []logr.Field, separator string) error {
	fields := make([]logr.Field, len(flds))
	copy(fields, flds)

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Key < fields[j].Key
	})

	errs := merror.New()
	objEncoder := encoder.NewObjectEncoder(w, separator)

	sep := ""
	for _, f := range fields {
		err := writeField(w, f, sep)
		if err != nil {
			errs.Append(err)
		}
		sep = separator
	}
	return errs.ErrorOrNil()
}

func writeField(w io.Writer, field Field, sep string) error {
	switch field.Type {
	case UnknownType:
		fmt.Fprintf(w, "%s:UnknownType", field.Key)
	case ArrayMarshalerType:
		marshaler, ok := field.Interface.(ArrayMarshaler)
		if !ok {
			return fmt.Errorf("invalid array marshaller for key %s", field.Key)
		}
		arrEncoder := newArrayEncoder(w, sep)
		if err := marshaler.MarshalLogArray(arrEncoder); err != nil {
			return fmt.Errorf("error marshalling array for key %s: %w", field.Key, err)
		}
	case ObjectMarshalerType:
		marshaler, ok := field.Interface.(ObjectMarshaler)
		if !ok {
			return fmt.Errorf("invalid object marshaller for key %s", field.Key)
		}
		objEncoder := newObjectEncoder(w, sep)
		if err := marshaler.MarshalLogObject(objEncoder); err != nil {
			return fmt.Errorf("error marshalling object for key %s: %w", field.Key, err)
		}
	case BinaryType:

	}
	/*
		case error:
			val := v.Error()
			if shouldQuote(val) {
				template = "%s%s=%q"
			} else {
				template = "%s%s=%s"
			}
		case string:
			if shouldQuote(v) {
				template = "%s%s=%q"
			} else {
				template = "%s%s=%s"
			}
		default:
			template = "%s%s=%v"
		}
		fmt.Fprintf(w, template, sep, key, val)
	*/
}

// ShouldQuote returns true if val contains any characters that might be unsafe
// when injecting log output into an aggregator, viewer or report.
func ShouldQuote(val string) bool {
	for _, c := range val {
		if !((c >= '0' && c <= '9') ||
			(c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z')) {
			return true
		}
	}
	return false
}

// WriteStacktrace formats and outputs a stack trace to an io.Writer.
func WriteStacktrace(w io.Writer, frames []runtime.Frame) {
	for _, frame := range frames {
		if frame.Function != "" {
			fmt.Fprintf(w, "  %s\n", frame.Function)
		}
		if frame.File != "" {
			fmt.Fprintf(w, "      %s:%d\n", frame.File, frame.Line)
		}
	}
}
