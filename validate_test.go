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

	t.Run("counts characters not bytes", func(t *testing.T) {
		s := "üüüüü"
		TruncateOptionText("opt", &s, 4)
		require.Equal(t, "üüüü", s)
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
