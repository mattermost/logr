package logr

import (
	"bytes"
)

// Formatter turns a LogRec into a formatted string.
type Formatter interface {
	// Format converts a log record to bytes. If buf is not nil then it will be
	// be filled with the formatted results, otherwise a new buffer will be allocated.
	Format(rec *LogRec, stacktrace bool, buf *bytes.Buffer) (*bytes.Buffer, error)
}

const (
	// DefTimestampFormat is the default time stamp format used by
	// Plain formatter and others.
	DefTimestampFormat = "2006-01-02 15:04:05.000 Z07:00"
)
