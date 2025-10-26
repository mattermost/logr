package logr

import "testing"

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
