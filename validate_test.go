package logr

import (
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
}
