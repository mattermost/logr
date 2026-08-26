package logr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Should NOT be quoted (safe characters)
		{"numbers", "0123456789", false},
		{"lowercase", "abcdefghijklmnopqrstuvwxyz", false},
		{"uppercase", "ABCDEFGHIJKLMNOPQRSTUVWXYZ", false},
		{"dash", "test-value", false},
		{"dot", "test.value", false},
		{"underscore", "test_value", false},
		{"slash", "test/value", false},
		{"at", "test@value", false},
		{"caret", "test^value", false},
		{"plus", "test+value", false},
		{"alphanumeric with safe chars", "test123-value_foo.bar/baz@example.com^a+b", false},

		// Should be quoted (unsafe characters)
		{"space", "hello world", true},
		{"tab", "hello\tworld", true},
		{"newline", "hello\nworld", true},
		{"quote", "hello\"world", true},
		{"single quote", "hello'world", true},
		{"comma", "hello,world", true},
		{"semicolon", "hello;world", true},
		{"pipe", "hello|world", true},
		{"ampersand", "hello&world", true},
		{"dollar", "hello$world", true},
		{"asterisk", "hello*world", true},
		{"parentheses", "hello(world)", true},
		{"brackets", "hello[world]", true},
		{"braces", "hello{world}", true},
		{"equals", "hello=world", true},
		{"exclamation", "hello!world", true},
		{"question", "hello?world", true},
		{"less than", "hello<world", true},
		{"greater than", "hello>world", true},
		{"backslash", "hello\\world", true},
		{"tilde", "hello~world", true},
		{"backtick", "hello`world", true},
		{"percent", "hello%world", true},
		{"hash", "hello#world", true},
		{"colon", "hello:world", true},

		// Edge cases
		{"empty string", "", false},
		{"single safe char", "a", false},
		{"single unsafe char", " ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldQuote(tt.input)
			if result != tt.expected {
				t.Errorf("shouldQuote(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

type formatterTestStringer struct {
	value string
}

func (ts *formatterTestStringer) String() string {
	return ts.value
}

func TestLimitedStringer_String(t *testing.T) {
	tests := []struct {
		name     string
		stringer fmt.Stringer
		limit    int
		expected string
	}{
		{
			name:     "string shorter than limit",
			stringer: &formatterTestStringer{value: "hello"},
			limit:    10,
			expected: "hello",
		},
		{
			name:     "string equal to limit",
			stringer: &formatterTestStringer{value: "hello"},
			limit:    5,
			expected: "hello",
		},
		{
			name:     "string longer than limit",
			stringer: &formatterTestStringer{value: "hello world"},
			limit:    5,
			expected: "hello...", // LimitString adds "..." when truncating
		},
		{
			name:     "empty string",
			stringer: &formatterTestStringer{value: ""},
			limit:    10,
			expected: "",
		},
		{
			name:     "zero limit",
			stringer: &formatterTestStringer{value: "hello"},
			limit:    0,
			expected: "hello", // limit <= 0 returns original string
		},
		{
			name:     "unicode characters",
			stringer: &formatterTestStringer{value: "hello 世界"},
			limit:    8,
			expected: "hello ...", // Truncates at UTF-8 character boundary and adds "..."
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := &LimitedStringer{
				Stringer: tt.stringer,
				Limit:    tt.limit,
			}
			result := ls.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultFormatter_IsStacktraceNeeded(t *testing.T) {
	formatter := &DefaultFormatter{}
	result := formatter.IsStacktraceNeeded()
	assert.False(t, result, "DefaultFormatter should not need stacktrace")
}
