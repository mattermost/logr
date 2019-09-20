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
	defTimestampFormat = "2006-01-02 15:04:05.000 Z07:00"
)

// Plain is the simplest formatter, outputting only text with
// no colors.
type Plain struct {
	DisableTimestamp  bool
	DisableLevel      bool
	DisableMsg        bool
	DisableFields     bool
	DisableStacktrace bool

	Delim           string
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
		timestampFmt = defTimestampFormat
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
		WriteFields(sb, rec.Fields(), ", ")
	}
	if !p.DisableStacktrace {
		WriteStacktrace(sb, rec.StackFrames())
	}
	sb.WriteString("\n")

	return []byte(sb.String()), nil
}

// WriteFields writes zero or more name value pairs to the io.Writer.
// The pairs are sorted by key name.
func WriteFields(w io.Writer, flds logr.Fields, separator string) {
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

func WriteStacktrace(w io.Writer, frames []runtime.Frame) {

}
