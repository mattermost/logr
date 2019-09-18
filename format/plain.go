package format

import (
	"fmt"
	"strings"
	"time"

	"github.com/wiggin77/logr"
)

// Plain is the simplest formatter, outputting only text with
// no colors.
type Plain struct {
	DisableTimestamp  bool
	DisableLevel      bool
	DisableMsg        bool
	DisableFields     bool
	DisableStacktrace bool
	Delim             string
}

// Format converts a log record to bytes.
func (p *Plain) Format(rec *logr.LogRec) ([]byte, error) {
	var sb strings.Builder

	delim := p.Delim
	if delim == "" {
		delim = " "
	}

	if !p.DisableTimestamp {
		fmt.Fprintf(&sb, "%s%s", rec.Time().Format(time.RFC3339), delim)
	}
	if !p.DisableLevel {
		fmt.Fprintf(&sb, "%v%s", rec.Level(), delim)
	}
	if !p.DisableMsg {
		fmt.Fprintf(&sb, "%s%s", rec.Msg(), delim)
	}
	sb.WriteString("\n")

	return []byte(sb.String()), nil
}
