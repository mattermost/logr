package logr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckOptionLen(t *testing.T) {
	require.NoError(t, CheckOptionLen("opt", "", 4))
	require.NoError(t, CheckOptionLen("opt", "abcd", 4))

	err := CheckOptionLen("opt", "abcde", 4)
	require.Error(t, err)
	require.Contains(t, err.Error(), "opt is too long")

	t.Run("counts characters not bytes", func(t *testing.T) {
		// 4 characters, 8 bytes. JSON Schema maxLength counts characters, so the
		// Go limit has to as well or the two disagree on the same value.
		require.NoError(t, CheckOptionLen("opt", "üüüü", 4))
		require.Error(t, CheckOptionLen("opt", "üüüüü", 4))
	})
}

func TestCheckOptionText(t *testing.T) {
	require.NoError(t, CheckOptionText("opt", "log | ", 32))
	require.NoError(t, CheckOptionText("opt", "ünïcøde", 32))
	require.NoError(t, CheckOptionText("opt", "日本語", 32))

	t.Run("allows tab", func(t *testing.T) {
		// A tab is a legitimate delimiter for TSV-style output and cannot
		// terminate a record or introduce an escape sequence.
		require.NoError(t, CheckOptionText("delim", "\t", 32))
		require.NoError(t, CheckOptionText("delim", "a\tb", 32))
	})

	t.Run("rejects newline", func(t *testing.T) {
		err := CheckOptionText("opt", "a\nb", 32)
		require.Error(t, err)
		require.Contains(t, err.Error(), "disallowed character")
		require.Contains(t, err.Error(), "U+000A")
	})

	t.Run("rejects C0 controls and DEL", func(t *testing.T) {
		require.Error(t, CheckOptionText("opt", "\x00", 32))
		require.Error(t, CheckOptionText("opt", "\x01", 32))
		require.Error(t, CheckOptionText("opt", "\x1f", 32))
		require.Error(t, CheckOptionText("opt", "a\x7fb", 32))
	})

	t.Run("rejects both ANSI escape introducers", func(t *testing.T) {
		require.Error(t, CheckOptionText("opt", "\x1b[31m", 32), "7-bit ESC")
		require.Error(t, CheckOptionText("opt", "\u009b[2J", 32), "8-bit CSI")
	})

	t.Run("rejects C1 controls", func(t *testing.T) {
		require.Error(t, CheckOptionText("opt", "a\u0085b", 32), "NEL is a line break to many log readers")
		require.Error(t, CheckOptionText("opt", "a\u0080b", 32))
		require.Error(t, CheckOptionText("opt", "a\u009fb", 32))
	})

	t.Run("rejects bidi controls", func(t *testing.T) {
		for _, r := range []rune{
			'\u200e', '\u200f', '\u202a', '\u202b', '\u202c', '\u202d', '\u202e',
			'\u2066', '\u2067', '\u2068', '\u2069',
		} {
			require.Error(t, CheckOptionText("opt", "a"+string(r)+"b", 32), "%U should be rejected", r)
		}
	})

	t.Run("rejects line and paragraph separators", func(t *testing.T) {
		require.Error(t, CheckOptionText("opt", "a\u2028b", 32))
		require.Error(t, CheckOptionText("opt", "a\u2029b", 32))
	})

	t.Run("rejects invalid UTF-8", func(t *testing.T) {
		err := CheckOptionText("opt", string([]byte{0xff, 0xfe}), 32)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not valid UTF-8")

		// An overlong encoding of ESC decodes to U+FFFD in a per-rune scan, so
		// it would slip past a check that only looked at decoded runes.
		require.Error(t, CheckOptionText("opt", string([]byte{0xc0, 0x9b}), 32))
		require.Error(t, CheckOptionText("opt", string([]byte{0x9b}), 32))
	})

	t.Run("rejects over length", func(t *testing.T) {
		require.Error(t, CheckOptionText("opt", strings.Repeat("a", 33), 32))
	})
}

func TestCheckOptionHost(t *testing.T) {
	require.NoError(t, CheckOptionHost("host", ""), "empty means local or unset")
	require.NoError(t, CheckOptionHost("host", "logs.example.com"))
	require.NoError(t, CheckOptionHost("host", "127.0.0.1"))
	require.NoError(t, CheckOptionHost("host", "::1"))

	t.Run("rejects whitespace", func(t *testing.T) {
		// A host is interpolated into "host:port" and dialed, so tab and space
		// produce an address that cannot resolve. Tab is valid for a delimiter,
		// which is why CheckOptionText alone is not enough here.
		for _, s := range []string{"log\tserver", "log server", "\tlogs", "logs "} {
			err := CheckOptionHost("host", s)
			require.Error(t, err, "%q should be rejected", s)
			require.Contains(t, err.Error(), "whitespace")
		}
	})

	t.Run("rejects control characters", func(t *testing.T) {
		require.Error(t, CheckOptionHost("host", "logs\n.example.com"))
	})

	t.Run("is length limited", func(t *testing.T) {
		require.Error(t, CheckOptionHost("host", strings.Repeat("h", MaxHostnameLen+1)))
	})
}

func TestCheckOptionLineEnd(t *testing.T) {
	for _, s := range []string{"", "\n", "\r\n", "\n\n", " \n", "\t\n"} {
		require.NoError(t, CheckOptionLineEnd("line_end", s), "%q should be valid", s)
	}

	t.Run("rejects payload", func(t *testing.T) {
		err := CheckOptionLineEnd("line_end", "#!/bin/sh\nid\n")
		require.Error(t, err)
		require.Contains(t, err.Error(), "only whitespace")
	})

	t.Run("requires a line break", func(t *testing.T) {
		// Whitespace with no break would run every record onto one line, so
		// individual records could not be recovered from the output.
		for _, s := range []string{" ", "  ", "\t", "\t\t", strings.Repeat(" ", MaxLineEndLen)} {
			err := CheckOptionLineEnd("line_end", s)
			require.Error(t, err, "%q should be rejected", s)
			require.Contains(t, err.Error(), "must contain a line break")
		}
	})

	t.Run("rejects over length", func(t *testing.T) {
		require.Error(t, CheckOptionLineEnd("line_end", strings.Repeat("\n", MaxLineEndLen+1)))
	})
}

func TestCheckOptionRange(t *testing.T) {
	require.NoError(t, CheckOptionRange("opt", 0, 10))
	require.NoError(t, CheckOptionRange("opt", 10, 10))
	require.Error(t, CheckOptionRange("opt", -1, 10))
	require.Error(t, CheckOptionRange("opt", 11, 10))
}

func TestLevelCheckValid(t *testing.T) {
	require.NoError(t, Level{ID: 1, Name: "info"}.CheckValid())
	require.NoError(t, Level{ID: MaxLevelID, Name: "custom"}.CheckValid())

	t.Run("rejects embedded newline", func(t *testing.T) {
		require.Error(t, Level{ID: 1, Name: "info\nforged line"}.CheckValid())
	})

	t.Run("rejects over length", func(t *testing.T) {
		require.Error(t, Level{ID: 1, Name: strings.Repeat("a", MaxLevelNameLen+1)}.CheckValid())
	})

	t.Run("rejects id above MaxLevelID", func(t *testing.T) {
		// An ID above MaxLevelID cannot be cached, so ReportError fires on every
		// log call at that level and the record is dropped.
		err := Level{ID: MaxLevelID + 1, Name: "custom"}.CheckValid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "MaxLevelID")
	})
}
