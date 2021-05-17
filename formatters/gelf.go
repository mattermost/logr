package formatters

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/francoispqt/gojay"
	"github.com/mattermost/logr/v2"
)

const (
	GelfVersion      = "1.1"
	GelfVersionKey   = "version"
	GelfHostKey      = "host"
	GelfShortKey     = "short_message"
	GelfFullKey      = "full_message"
	GelfTimestampKey = "timestamp"
	GelfLevelKey     = "level"
)

// Gelf formats log records as GELF rcords (https://docs.graylog.org/en/4.0/pages/gelf.html).
type Gelf struct {
	// Hostname allows a custom hostname, otherwise os.Hostname is used
	Hostname string

	// FieldSorter allows custom sorting for the context fields.
	FieldSorter func(fields []logr.Field) []logr.Field
}

// Format converts a log record to bytes in GELF format.
func (g *Gelf) Format(rec *logr.LogRec, stacktrace bool, buf *bytes.Buffer) (*bytes.Buffer, error) {
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	enc := gojay.BorrowEncoder(buf)
	defer func() {
		enc.Release()
	}()

	gr := gelfRecord{
		LogRec:     rec,
		Gelf:       g,
		stacktrace: stacktrace,
		sorter:     g.FieldSorter,
	}

	err := enc.EncodeObject(gr)
	if err != nil {
		return nil, err
	}

	buf.WriteByte(0)
	return buf, nil
}

type gelfRecord struct {
	*logr.LogRec
	*Gelf
	stacktrace bool
	sorter     func(fields []logr.Field) []logr.Field
}

// MarshalJSONObject encodes the LogRec as JSON.
func (gr gelfRecord) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddStringKey(GelfVersionKey, GelfVersion)
	enc.AddStringKey(GelfHostKey, gr.getHostname())
	enc.AddStringKey(GelfShortKey, gr.Msg())

	if gr.stacktrace {
		frames := gr.StackFrames()
		if len(frames) != 0 {
			var sbuf strings.Builder
			for _, frame := range frames {
				fmt.Fprintf(&sbuf, "%s\n  %s:%d\n", frame.Function, frame.File, frame.Line)
			}
			enc.AddStringKey(GelfFullKey, sbuf.String())
		}
	}

	secs := float64(gr.Time().UTC().Unix())
	millis := float64(gr.Time().Nanosecond() / 1000000)
	ts := secs + (millis / 1000)
	enc.AddFloat64Key(GelfTimestampKey, ts)

	enc.AddUint32Key(GelfLevelKey, uint32(gr.Level().ID))

	fields := gr.Fields()
	if gr.sorter != nil {
		fields = gr.sorter(fields)
	}

	if len(fields) > 0 {
		for _, field := range fields {
			if !strings.HasPrefix("_", field.Key) {
				field.Key = "_" + field.Key
			}
			if err := encodeField(enc, field); err != nil {
				enc.AddStringKey(field.Key, fmt.Sprintf("<error encoding field: %v>", err))
			}
		}
	}
}

// IsNil returns true if the gelf record pointer is nil.
func (gr gelfRecord) IsNil() bool {
	return gr.LogRec == nil
}

func (g *Gelf) getHostname() string {
	if g.Hostname != "" {
		return g.Hostname
	}
	h, err := os.Hostname()
	if err == nil {
		return h
	}

	// get the egress IP by fake dialing any address. UDP ensures no dial.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "unknown"
	}
	defer conn.Close()

	local := conn.LocalAddr().(*net.UDPAddr)
	return local.IP.String()
}
