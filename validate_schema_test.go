package logr

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// The limits in validate.go and the constraints in logr-config-schema.json are
// two expressions of the same rules: the Go checks are what logr enforces, the
// schema is what external validators enforce. These tests fail if the two drift
// apart.
//
// The invariant is an implication, not an equality: anything a Go check accepts
// the schema must also accept, so a config that logr applies never fails
// external validation. The Go checks are free to be stricter, and are.
//
// Schema patterns are ECMA-262 regexes, which spell codepoints above U+00FF as
// \uXXXX while Go regexp wants \x{XXXX}. goPattern translates between them so
// the schema can use standard JSON Schema escapes.

const schemaPath = "logr-config-schema.json"

type schemaConstraint struct {
	MaxLength *int   `json:"maxLength"`
	Minimum   *int   `json:"minimum"`
	Maximum   *int   `json:"maximum"`
	Pattern   string `json:"pattern"`
}

type schemaDoc struct {
	Definitions map[string]struct {
		Properties map[string]schemaConstraint `json:"properties"`
	} `json:"definitions"`
}

// loadSchema reads and parses the schema once for the whole test binary.
var loadSchema = sync.OnceValue(func() schemaDoc {
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		panic(fmt.Sprintf("cannot read %s: %v", schemaPath, err))
	}

	var doc schemaDoc
	if err := json.Unmarshal(b, &doc); err != nil {
		panic(fmt.Sprintf("cannot parse %s: %v", schemaPath, err))
	}
	return doc
})

// goPattern converts an ECMA-262 pattern to Go regexp syntax, which differs
// only in how it spells codepoints above U+00FF.
func goPattern(pattern string) string {
	return regexp.MustCompile(`\\u([0-9a-fA-F]{4})`).ReplaceAllString(pattern, `\x{$1}`)
}

// compilePattern compiles a schema pattern for use from Go.
func compilePattern(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()

	re, err := regexp.Compile(goPattern(pattern))
	require.NoError(t, err, "schema pattern %q should compile", pattern)
	return re
}

// loadSchemaConstraints returns the constraints for a property of a definition,
// e.g. ("plainFormatOptions", "delim").
func loadSchemaConstraints(t *testing.T, definition string, property string) schemaConstraint {
	t.Helper()

	def, ok := loadSchema().Definitions[definition]
	require.True(t, ok, "schema should define %s", definition)

	c, ok := def.Properties[property]
	require.True(t, ok, "%s should define property %s", definition, property)
	return c
}

// corpus covers the character classes the limits are meant to separate.
var corpus = []string{
	"",
	" ",
	"  ",
	"\n",
	"\r\n",
	"\t",
	"\n\n",
	" \n",
	"|",
	" | ",
	" - ",
	"info",
	"/opt/mattermost/logs/mattermost.log",
	"ünïcøde",
	"日本語のデリミタ",
	"\x00",
	"a\x00b",
	"\x01",
	"\x1f",
	"\x7f",
	"a\x7fb",
	"\x1b[31m",
	"a\nb",
	"#!/bin/sh\nid\n",
	"line\rreturn",
	"tab\there",
	"\u0085",                   // NEL, a C1 control
	"\u009b[2J",                // CSI, the 8-bit form of ESC [
	"a\u202eb",                 // RIGHT-TO-LEFT OVERRIDE
	"a\u2028b",                 // LINE SEPARATOR
	"a\u2029b",                 // PARAGRAPH SEPARATOR
	string([]byte{0xff, 0xfe}), // invalid UTF-8
	string([]byte{0x9b}),       // raw C1 byte, invalid UTF-8
	string([]byte{0xc0, 0x9b}), // overlong encoding of ESC
}

// schemaAccepts reports whether an external JSON Schema validator would accept
// s for the given constraints. maxLength counts characters, per the JSON Schema
// spec — not bytes, which is what Go's len() would give.
func schemaAccepts(c schemaConstraint, re *regexp.Regexp, s string) bool {
	if c.MaxLength != nil && utf8.RuneCountInString(s) > *c.MaxLength {
		return false
	}
	return re.MatchString(s)
}

func TestSchemaMaxLengthsMatchGoLimits(t *testing.T) {
	cases := []struct {
		definition string
		property   string
		goLimit    int
	}{
		{"level", "name", MaxLevelNameLen},
		{"plainFormatOptions", "delim", MaxDelimLen},
		{"plainFormatOptions", "line_end", MaxLineEndLen},
		{"plainFormatOptions", "timestamp_format", MaxTimestampFormatLen},
		{"jsonFormatOptions", "timestamp_format", MaxTimestampFormatLen},
		{"jsonFormatOptions", "key_timestamp", MaxFieldKeyLen},
		{"jsonFormatOptions", "key_level", MaxFieldKeyLen},
		{"jsonFormatOptions", "key_msg", MaxFieldKeyLen},
		{"jsonFormatOptions", "key_group_fields", MaxFieldKeyLen},
		{"jsonFormatOptions", "key_stacktrace", MaxFieldKeyLen},
		{"jsonFormatOptions", "key_caller", MaxFieldKeyLen},
		{"gelfFormatOptions", "hostname", MaxHostnameLen},
		{"fileOptions", "filename", MaxFilePathLen},
		{"tcpOptions", "host", MaxHostnameLen},
		{"tcpOptions", "ip", MaxHostnameLen},
		{"tcpOptions", "cert", MaxCertLen},
		{"syslogOptions", "host", MaxHostnameLen},
		{"syslogOptions", "ip", MaxHostnameLen},
		{"syslogOptions", "cert", MaxCertLen},
		{"syslogOptions", "tag", MaxTagLen},
	}

	for _, tc := range cases {
		t.Run(tc.definition+"."+tc.property, func(t *testing.T) {
			c := loadSchemaConstraints(t, tc.definition, tc.property)
			require.NotNil(t, c.MaxLength, "schema should set maxLength")
			require.Equal(t, tc.goLimit, *c.MaxLength, "schema maxLength should match the Go limit")
		})
	}
}

func TestSchemaNumericBoundsMatchGoLimits(t *testing.T) {
	cases := []struct {
		definition string
		property   string
		min        int
		max        int
		why        string
	}{
		{"plainFormatOptions", "min_level_len", 0, MaxPadLen, "CheckOptionRange in Plain.CheckValid"},
		{"plainFormatOptions", "min_msg_len", 0, MaxPadLen, "CheckOptionRange in Plain.CheckValid"},
		{"level", "id", 0, MaxLevelID, "Level.CheckValid bounds ID by MaxLevelID"},
		{"tcpOptions", "port", 1, 65535, "TcpOptions.CheckValid rejects port <= 0"},
		{"syslogOptions", "port", 0, 65535, "SyslogOptions.CheckValid allows port 0 for local syslog"},
	}

	for _, tc := range cases {
		t.Run(tc.definition+"."+tc.property, func(t *testing.T) {
			c := loadSchemaConstraints(t, tc.definition, tc.property)

			require.NotNil(t, c.Minimum, "schema should set minimum (%s)", tc.why)
			require.Equal(t, tc.min, *c.Minimum, "schema minimum should match Go (%s)", tc.why)

			require.NotNil(t, c.Maximum, "schema should set maximum (%s)", tc.why)
			require.Equal(t, tc.max, *c.Maximum, "schema maximum should match Go (%s)", tc.why)
		})
	}
}

// TestGoChecksAreAtLeastAsStrictAsSchema compiles each schema pattern and
// requires that everything the Go check accepts is also accepted by the schema.
// The reverse does not hold: see the note at the top of this file.
func TestGoChecksAreAtLeastAsStrictAsSchema(t *testing.T) {
	cases := []struct {
		definition string
		property   string
		check      func(s string) error
	}{
		{
			definition: "plainFormatOptions",
			property:   "delim",
			check:      func(s string) error { return CheckOptionText("delim", s, MaxDelimLen) },
		},
		{
			definition: "plainFormatOptions",
			property:   "line_end",
			check:      func(s string) error { return CheckOptionLineEnd("line_end", s) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.definition+"."+tc.property, func(t *testing.T) {
			c := loadSchemaConstraints(t, tc.definition, tc.property)
			require.NotEmpty(t, c.Pattern, "schema should set a pattern")

			re := compilePattern(t, c.Pattern)

			for _, s := range corpus {
				if tc.check(s) == nil {
					require.True(t, schemaAccepts(c, re, s),
						"Go accepts %q but the schema rejects it, so a valid config would fail external validation", s)
				}
			}
		})
	}
}

// TestGoChecksRejectBeyondSchemaPatterns pins the classes the Go checks reject
// that a schema pattern is not required to cover. It deliberately asserts
// nothing about what the schema accepts, so the schema is free to get stricter.
func TestGoChecksRejectBeyondSchemaPatterns(t *testing.T) {
	cases := []struct {
		name string
		val  string
	}{
		{"bidi override", "a\u202eb"},
		{"arabic letter mark", "a\u061cb"},
		{"line separator", "a\u2028b"},
		{"paragraph separator", "a\u2029b"},
		{"invalid UTF-8", string([]byte{0xff, 0xfe})},
		{"overlong ESC", string([]byte{0xc0, 0x9b})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, CheckOptionText("delim", tc.val, MaxDelimLen), "Go should reject %q", tc.val)
		})
	}
}

// TestSchemaPatternsRejectPayloads pins the property that motivates the
// patterns: neither the schema nor the Go check may accept a value that can
// carry arbitrary content into a log target.
func TestSchemaPatternsRejectPayloads(t *testing.T) {
	payloads := []string{
		"#!/bin/sh\nid\n",
		"\nforged log line ",
		"\x1b[2J",
	}

	lineEnd := loadSchemaConstraints(t, "plainFormatOptions", "line_end")
	lineEndRe := compilePattern(t, lineEnd.Pattern)

	delim := loadSchemaConstraints(t, "plainFormatOptions", "delim")
	delimRe := compilePattern(t, delim.Pattern)

	for _, payload := range payloads {
		require.Error(t, CheckOptionLineEnd("line_end", payload), "Go should reject %q for line_end", payload)
		require.Error(t, CheckOptionText("delim", payload, MaxDelimLen), "Go should reject %q for delim", payload)

		require.False(t, schemaAccepts(lineEnd, lineEndRe, payload), "schema should reject %q for line_end", payload)
		require.False(t, schemaAccepts(delim, delimRe, payload), "schema should reject %q for delim", payload)
	}
}

// TestSchemaPatternsAreGoCompatible checks every pattern in the schema is a
// regex these tests can evaluate, so a malformed one is caught here rather than
// by whatever tries to consume it.
func TestSchemaPatternsAreGoCompatible(t *testing.T) {
	doc := loadSchema()

	var found int
	for defName, def := range doc.Definitions {
		for propName, c := range def.Properties {
			if c.Pattern == "" {
				continue
			}
			found++
			t.Run(defName+"."+propName, func(t *testing.T) {
				compilePattern(t, c.Pattern)
			})
		}
	}
	require.NotZero(t, found, "schema should contain at least one pattern")
}

// TestSchemaBoundsFileRotationFields covers the file rotation options, which
// are bounded below only, so TestSchemaNumericBoundsMatchGoLimits (which
// asserts a maximum too) cannot cover them. Every other validated field is
// covered by the two tables above.
func TestSchemaBoundsFileRotationFields(t *testing.T) {
	for _, property := range []string{"max_size", "max_age", "max_backups"} {
		t.Run(property, func(t *testing.T) {
			c := loadSchemaConstraints(t, "fileOptions", property)
			require.NotNil(t, c.Minimum, "FileOptions.CheckValid rejects a negative %s", property)
			require.Zero(t, *c.Minimum, "Go accepts %s of 0, so the schema must too", property)
		})
	}
}
