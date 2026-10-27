package logr

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncateOptionText(t *testing.T) {
	s := "abcd"
	TruncateOptionText("opt", &s, 4)
	require.Equal(t, "abcd", s, "a value within the limit is left untouched")

	s = "abcde"
	TruncateOptionText("opt", &s, 4)
	require.Equal(t, "abcd", s)

	t.Run("counts bytes and never splits a character", func(t *testing.T) {
		s := "üüü"
		TruncateOptionText("opt", &s, 5)
		require.Equal(t, "üü", s)

		s = "ü"
		TruncateOptionText("opt", &s, 1)
		require.Equal(t, "", s)
	})

	t.Run("reports through TruncateWriter instead of a hardcoded stream", func(t *testing.T) {
		orig := TruncateWriter
		t.Cleanup(func() { TruncateWriter = orig })

		var buf bytes.Buffer
		TruncateWriter = &buf

		s := "abcde"
		TruncateOptionText("opt", &s, 4)
		require.Contains(t, buf.String(), "opt is too long")
	})

	t.Run("a nil TruncateWriter is a no-op instead of a panic", func(t *testing.T) {
		orig := TruncateWriter
		t.Cleanup(func() { TruncateWriter = orig })
		TruncateWriter = nil

		s := "abcde"
		require.NotPanics(t, func() { TruncateOptionText("opt", &s, 4) })
		require.Equal(t, "abcd", s)
	})
}

func TestCheckOptionText(t *testing.T) {
	require.NoError(t, CheckOptionText("opt", ""))
	require.NoError(t, CheckOptionText("opt", "info"))
	require.NoError(t, CheckOptionText("opt", "2006-01-02 15:04:05.000 Z07:00"))
	require.NoError(t, CheckOptionText("opt", "ünïcode"))

	for _, s := range []string{"a\nb", "a\rb", "a\tb", "\x1b[31m", "a\x00b", "a\x7fb", "a\u0085b", "a b", "a b"} {
		require.Error(t, CheckOptionText("opt", s), "%q should be rejected", s)
	}
}
