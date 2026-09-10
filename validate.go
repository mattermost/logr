package logr

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Maximum lengths for configuration-supplied string options, counted in
// characters to match the `maxLength` keyword in logr-config-schema.json.
//
// Log configuration is frequently sourced from a remote or delegated
// administrative surface, so every option that is copied verbatim into log
// output — or that is used to reach the filesystem or network — is bounded.
// Without these bounds an option such as a line terminator or field delimiter
// is an arbitrary-content write primitive for whatever the target points at.
const (
	// MaxLevelNameLen is the maximum length of a Level name.
	MaxLevelNameLen = 64

	// MaxFieldKeyLen is the maximum length of a formatter field key name.
	MaxFieldKeyLen = 64

	// MaxTimestampFormatLen is the maximum length of a timestamp format string.
	MaxTimestampFormatLen = 128

	// MaxDelimLen is the maximum length of a field delimiter.
	MaxDelimLen = 32

	// MaxLineEndLen is the maximum length of a line terminator.
	MaxLineEndLen = 16

	// MaxPadLen is the maximum padding width for a formatter field.
	MaxPadLen = 1024

	// MaxHostnameLen is the maximum length of a hostname option, matching the
	// maximum length of a DNS name.
	MaxHostnameLen = 255

	// MaxTagLen is the maximum length of a syslog tag.
	MaxTagLen = 64

	// MaxFilePathLen is the maximum length of a file path option.
	MaxFilePathLen = 4096

	// MaxCertLen is the maximum length of a cert option, which can hold either a
	// path or a base64 encoded cert chain.
	MaxCertLen = 64 * 1024
)

// Validator is implemented by option types that can check their own values.
// Targets and formatters supplied through `config.Factories` are validated if
// they implement it, so custom implementations are held to the same limits as
// the built-in ones.
type Validator interface {
	CheckValid() error
}

// isForbiddenInText reports whether r must not appear in a
// configuration-supplied string that is written to log output.
//
// Control characters allow a config value to forge additional log records or
// emit terminal escape sequences: U+001B and U+009B are both introducers for
// ANSI escapes, and U+000A and U+0085 both start a new line for most log
// readers. Bidirectional controls and the line and paragraph separators let a
// value misrepresent how the surrounding record reads.
//
// Tab is permitted: it is a legitimate field delimiter and cannot terminate a
// record or introduce an escape sequence.
func isForbiddenInText(r rune) bool {
	if r == '\t' {
		return false
	}
	return unicode.IsControl(r) || unicode.In(r, unicode.Bidi_Control, unicode.Zl, unicode.Zp)
}

// CheckOptionLen returns an error if s is longer than maxLen characters.
// Length is counted in characters rather than bytes so that the limit matches
// the `maxLength` keyword in logr-config-schema.json.
func CheckOptionLen(name string, s string, maxLen int) error {
	if count := utf8.RuneCountInString(s); count > maxLen {
		return fmt.Errorf("%s is too long (%d characters, maximum %d)", name, count, maxLen)
	}
	return nil
}

// CheckOptionText returns an error if s is longer than maxLen characters, is not
// valid UTF-8, or contains a character that must not reach log output. See
// `isForbiddenInText` for what is rejected and why.
func CheckOptionText(name string, s string, maxLen int) error {
	if err := CheckOptionLen(name, s, maxLen); err != nil {
		return err
	}
	if !utf8.ValidString(s) {
		// Invalid UTF-8 can smuggle a forbidden character past a per-rune scan,
		// for example an overlong encoding of U+001B.
		return fmt.Errorf("%s is not valid UTF-8", name)
	}
	for i, r := range s {
		if isForbiddenInText(r) {
			return fmt.Errorf("%s contains a disallowed character (%U at offset %d)", name, r, i)
		}
	}
	return nil
}

// CheckOptionHost returns an error if s is not usable as a hostname. It is
// stricter than CheckOptionText, which permits tab so it can serve as a field
// delimiter: a host is interpolated into a "host:port" address and dialed, so
// whitespace in it produces an address that cannot resolve.
func CheckOptionHost(name string, s string) error {
	if err := CheckOptionText(name, s, MaxHostnameLen); err != nil {
		return err
	}
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return fmt.Errorf("%s must not contain whitespace (at offset %d)", name, i)
	}
	return nil
}

// lineEndChars are the only characters permitted in a line terminator option.
const lineEndChars = "\r\n\t "

// CheckOptionLineEnd returns an error if s is longer than MaxLineEndLen,
// contains anything other than whitespace, or contains no line break at all.
//
// A line terminator is written after every log record, so unrestricted content
// there is enough to write an arbitrary payload to a target regardless of what
// is logged. Requiring a line break keeps records separable: a terminator of
// only spaces or tabs would run every record onto one unbounded line.
//
// An empty value is accepted and means "use the default", which formatters
// resolve to "\n".
func CheckOptionLineEnd(name string, s string) error {
	if s == "" {
		return nil
	}
	if err := CheckOptionLen(name, s, MaxLineEndLen); err != nil {
		return err
	}
	if i := strings.IndexFunc(s, func(r rune) bool { return !strings.ContainsRune(lineEndChars, r) }); i >= 0 {
		return fmt.Errorf("%s must contain only whitespace (invalid character at offset %d)", name, i)
	}
	if !strings.ContainsAny(s, "\r\n") {
		return fmt.Errorf("%s must contain a line break", name)
	}
	return nil
}

// CheckOptionRange returns an error if val is outside [0, maxVal].
func CheckOptionRange(name string, val int, maxVal int) error {
	if val < 0 || val > maxVal {
		return fmt.Errorf("%s is invalid (%d), must be between 0 and %d", name, val, maxVal)
	}
	return nil
}

// CheckValid returns an error if this level cannot be safely used. The level
// name is written verbatim to targets by most formatters, and an ID above
// MaxLevelID cannot be cached, which makes every record at that level fail.
func (level Level) CheckValid() error {
	if level.ID > MaxLevelID {
		return fmt.Errorf("level id is invalid (%d), must not exceed MaxLevelID (%d)", level.ID, MaxLevelID)
	}
	return CheckOptionText("level name", level.Name, MaxLevelNameLen)
}
