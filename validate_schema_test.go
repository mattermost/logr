package logr

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// The limits in validate.go and the constraints in logr-config-schema.json are
// two expressions of the same rules: the Go checks are what logr enforces, the
// schema is what external validators enforce. These tests fail if the two drift
// apart.
//
// The Go checks are deliberately stricter than the schema patterns. A JSON
// Schema pattern is an ECMA-262 regex, which expresses codepoints above U+00FF
// as \uXXXX, while Go regexp requires \x{XXXX} — there is no syntax both accept.
// The patterns therefore use \xHH escapes only and cover the C0, DEL and C1
// ranges, and the Go checks additionally reject bidi controls, the Unicode line
// and paragraph separators, and invalid UTF-8. So the invariant these tests hold
// is an implication, not an equality: anything the Go check accepts, the schema
// must also accept.

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

func loadSchema(t *testing.T) schemaDoc {
	t.Helper()

	b, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "should read %s", schemaPath)

	var doc schemaDoc
	require.NoError(t, json.Unmarshal(b, &doc))
	return doc
}

// loadSchemaConstraints returns the constraints for a property of a definition,
// e.g. ("plainFormatOptions", "delim").
func loadSchemaConstraints(t *testing.T, definition string, property string) schemaConstraint {
	t.Helper()

	doc := loadSchema(t)

	def, ok := doc.Definitions[definition]
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

			re, err := regexp.Compile(c.Pattern)
			require.NoError(t, err, "pattern %q should compile with Go regexp; use \\xHH escapes, not \\uXXXX", c.Pattern)

			for _, s := range corpus {
				if tc.check(s) == nil {
					require.True(t, schemaAccepts(c, re, s),
						"Go accepts %q but the schema rejects it, so a valid config would fail external validation", s)
				}
			}
		})
	}
}

// TestGoChecksRejectWhatSchemaPatternsCannotExpress pins the classes the Go
// checks catch beyond the schema patterns, so the gap stays deliberate.
func TestGoChecksRejectWhatSchemaPatternsCannotExpress(t *testing.T) {
	beyondSchema := []struct {
		name string
		val  string
	}{
		{"bidi override", "a\u202eb"},
		{"line separator", "a\u2028b"},
		{"paragraph separator", "a\u2029b"},
		{"invalid UTF-8", string([]byte{0xff, 0xfe})},
		{"overlong ESC", string([]byte{0xc0, 0x9b})},
	}

	delim := loadSchemaConstraints(t, "plainFormatOptions", "delim")
	delimRe := regexp.MustCompile(delim.Pattern)

	for _, tc := range beyondSchema {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, CheckOptionText("delim", tc.val, MaxDelimLen), "Go should reject %q", tc.val)
			require.True(t, schemaAccepts(delim, delimRe, tc.val),
				"this case exists because the schema pattern cannot express it; if the schema now rejects %q, tighten this test", tc.val)
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
		"\u009b[2J",
	}

	lineEnd := loadSchemaConstraints(t, "plainFormatOptions", "line_end")
	lineEndRe := regexp.MustCompile(lineEnd.Pattern)

	delim := loadSchemaConstraints(t, "plainFormatOptions", "delim")
	delimRe := regexp.MustCompile(delim.Pattern)

	for _, payload := range payloads {
		require.Error(t, CheckOptionLineEnd("line_end", payload), "Go should reject %q for line_end", payload)
		require.Error(t, CheckOptionText("delim", payload, MaxDelimLen), "Go should reject %q for delim", payload)

		require.False(t, schemaAccepts(lineEnd, lineEndRe, payload), "schema should reject %q for line_end", payload)

		if payload != "\u009b[2J" {
			// The C1 form is covered by the \x7f-\x9f range only when the value
			// is a raw byte; as UTF-8 it encodes to two bytes that the pattern
			// cannot address. TestGoChecksRejectWhatSchemaPatternsCannotExpress
			// records that gap.
			require.False(t, schemaAccepts(delim, delimRe, payload), "schema should reject %q for delim", payload)
		}
	}
}

// TestSchemaPatternsAreGoCompatible guards the escape syntax across every
// pattern in the schema, so a future pattern using \uXXXX is caught here
// rather than by whatever tries to consume it.
func TestSchemaPatternsAreGoCompatible(t *testing.T) {
	doc := loadSchema(t)

	var found int
	for defName, def := range doc.Definitions {
		for propName, c := range def.Properties {
			if c.Pattern == "" {
				continue
			}
			found++
			t.Run(defName+"."+propName, func(t *testing.T) {
				require.NotContains(t, c.Pattern, `\u`,
					`pattern should use \xHH escapes; \uXXXX is valid JSON Schema but Go regexp cannot compile it`)
				_, err := regexp.Compile(c.Pattern)
				require.NoError(t, err, "pattern %q should compile with Go regexp", c.Pattern)
			})
		}
	}
	require.NotZero(t, found, "schema should contain at least one pattern")
}

// TestSchemaConstrainsEveryValidatedField fails when a field that validate.go
// bounds has no corresponding constraint in the schema. Asserting the
// constraint exists, rather than just the property name, is what makes this
// catch a stripped maxLength.
func TestSchemaConstrainsEveryValidatedField(t *testing.T) {
	doc := loadSchema(t)

	stringFields := map[string][]string{
		"level":              {"name"},
		"plainFormatOptions": {"delim", "line_end", "timestamp_format"},
		"jsonFormatOptions": {
			"timestamp_format", "key_timestamp", "key_level", "key_msg",
			"key_group_fields", "key_stacktrace", "key_caller",
		},
		"gelfFormatOptions": {"hostname"},
		"fileOptions":       {"filename"},
		"tcpOptions":        {"host", "ip", "cert"},
		"syslogOptions":     {"host", "ip", "cert", "tag"},
	}

	numericFields := map[string][]string{
		"level":              {"id"},
		"plainFormatOptions": {"min_level_len", "min_msg_len"},
		"fileOptions":        {"max_size", "max_age", "max_backups"},
		"tcpOptions":         {"port"},
		"syslogOptions":      {"port"},
	}

	for defName, properties := range stringFields {
		def, ok := doc.Definitions[defName]
		require.True(t, ok, "schema should define %s", defName)

		for _, property := range properties {
			t.Run(defName+"."+property, func(t *testing.T) {
				c, ok := def.Properties[property]
				require.True(t, ok, "schema should describe %s", property)
				require.NotNil(t, c.MaxLength, "validate.go bounds the length of %s, so the schema needs maxLength", property)
			})
		}
	}

	for defName, properties := range numericFields {
		def, ok := doc.Definitions[defName]
		require.True(t, ok, "schema should define %s", defName)

		for _, property := range properties {
			t.Run(defName+"."+property, func(t *testing.T) {
				c, ok := def.Properties[property]
				require.True(t, ok, "schema should describe %s", property)
				require.NotNil(t, c.Minimum, "validate.go rejects negative %s, so the schema needs minimum", property)
			})
		}
	}
}
