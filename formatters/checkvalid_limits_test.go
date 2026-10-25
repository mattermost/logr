package formatters_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/stretchr/testify/require"
)

func TestPlainCheckValidLimits(t *testing.T) {
	t.Run("defaults are valid", func(t *testing.T) {
		require.NoError(t, (&formatters.Plain{}).CheckValid())
	})

	t.Run("typical options are valid", func(t *testing.T) {
		p := &formatters.Plain{
			Delim:           " | ",
			LineEnd:         "\r\n",
			MinLevelLen:     5,
			MinMessageLen:   40,
			TimestampFormat: logr.DefTimestampFormat,
		}
		require.NoError(t, p.CheckValid())
	})

	t.Run("line_end over length is truncated", func(t *testing.T) {
		p := &formatters.Plain{LineEnd: strings.Repeat("\n", logr.MaxLineEndLen+1)}
		require.NoError(t, p.CheckValid())
		require.Equal(t, strings.Repeat("\n", logr.MaxLineEndLen), p.LineEnd)
	})

	t.Run("delim over length is truncated", func(t *testing.T) {
		p := &formatters.Plain{Delim: strings.Repeat("x", logr.MaxDelimLen+1)}
		require.NoError(t, p.CheckValid())
		require.Equal(t, strings.Repeat("x", logr.MaxDelimLen), p.Delim)
	})

	t.Run("min_msg_len is bounded", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{MinMessageLen: -1}).CheckValid())
		require.Error(t, (&formatters.Plain{MinMessageLen: 1025}).CheckValid())
	})

	t.Run("timestamp_format over length is truncated with one warning", func(t *testing.T) {
		buf := captureTruncateWriter(t)
		p := &formatters.Plain{TimestampFormat: strings.Repeat("x", logr.MaxTimestampFormatLen+1)}
		require.NoError(t, p.CheckValid())
		require.Equal(t, strings.Repeat("x", logr.MaxTimestampFormatLen), p.TimestampFormat)
		require.Equal(t, 1, strings.Count(buf.String(), "\n"))
	})

	t.Run("timestamp_format with control characters is rejected", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{TimestampFormat: "2006]\nERROR [2006-01-02] fake\n["}).CheckValid())
		require.Error(t, (&formatters.Plain{TimestampFormat: "\x1b[31m2006"}).CheckValid())
	})

	t.Run("min_level_len over cap is clamped with one warning", func(t *testing.T) {
		buf := captureTruncateWriter(t)
		p := &formatters.Plain{MinLevelLen: logr.MaxLevelNameLen + 1}
		require.NoError(t, p.CheckValid())
		require.Equal(t, logr.MaxLevelNameLen, p.MinLevelLen)
		require.Equal(t, 1, strings.Count(buf.String(), "\n"))
	})

	t.Run("values at the cap are unchanged without a warning", func(t *testing.T) {
		buf := captureTruncateWriter(t)
		ts := strings.Repeat("x", logr.MaxTimestampFormatLen)
		p := &formatters.Plain{TimestampFormat: ts, MinLevelLen: logr.MaxLevelNameLen}
		require.NoError(t, p.CheckValid())
		require.Equal(t, ts, p.TimestampFormat)
		require.Equal(t, logr.MaxLevelNameLen, p.MinLevelLen)
		require.Empty(t, buf.String())
	})
}

func TestJSONCheckValidLimits(t *testing.T) {
	t.Run("timestamp_format over length is truncated with one warning", func(t *testing.T) {
		buf := captureTruncateWriter(t)
		j := &formatters.JSON{TimestampFormat: strings.Repeat("x", logr.MaxTimestampFormatLen+1)}
		require.NoError(t, j.CheckValid())
		require.Equal(t, strings.Repeat("x", logr.MaxTimestampFormatLen), j.TimestampFormat)
		require.Equal(t, 1, strings.Count(buf.String(), "\n"))
	})

	t.Run("default timestamp format is unchanged without a warning", func(t *testing.T) {
		buf := captureTruncateWriter(t)
		j := &formatters.JSON{TimestampFormat: logr.DefTimestampFormat}
		require.NoError(t, j.CheckValid())
		require.Equal(t, logr.DefTimestampFormat, j.TimestampFormat)
		require.Empty(t, buf.String())
	})
}

func captureTruncateWriter(t *testing.T) *bytes.Buffer {
	t.Helper()
	orig := logr.TruncateWriter
	t.Cleanup(func() { logr.TruncateWriter = orig })
	buf := &bytes.Buffer{}
	logr.TruncateWriter = buf
	return buf
}
