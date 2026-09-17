package formatters_test

import (
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

	t.Run("line_end over length is truncated instead of rejected", func(t *testing.T) {
		p := &formatters.Plain{LineEnd: strings.Repeat("\n", logr.MaxLineEndLen+1)}
		require.NoError(t, p.CheckValid())
		require.Equal(t, strings.Repeat("\n", logr.MaxLineEndLen), p.LineEnd)
	})

	t.Run("delim over length is truncated instead of rejected", func(t *testing.T) {
		p := &formatters.Plain{Delim: strings.Repeat("x", logr.MaxDelimLen+1)}
		require.NoError(t, p.CheckValid())
		require.Equal(t, strings.Repeat("x", logr.MaxDelimLen), p.Delim)
	})

	t.Run("min_msg_len is bounded", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{MinMessageLen: -1}).CheckValid())
		require.Error(t, (&formatters.Plain{MinMessageLen: 1025}).CheckValid())
	})

	t.Run("min_level_len is bounded", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{MinLevelLen: -1}).CheckValid())
		require.Error(t, (&formatters.Plain{MinLevelLen: 1025}).CheckValid())
	})
}
