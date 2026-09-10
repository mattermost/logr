package config

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/require"
)

// newLgr returns a logger that is shut down when the test ends.
func newLgr(t *testing.T) *logr.Logr {
	t.Helper()

	lgr, err := logr.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = lgr.Shutdown()
	})
	return lgr
}

// parseCfg deserializes a target config from raw JSON.
func parseCfg(t *testing.T, raw string) map[string]TargetCfg {
	t.Helper()

	var cfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))
	return cfg
}

// configureFromJSON builds a single-target config from raw JSON and applies it.
func configureFromJSON(t *testing.T, raw string) error {
	t.Helper()

	return ConfigureTargets(newLgr(t), parseCfg(t, raw), nil)
}

// fileTargetCfg returns a config for one valid file target, with formatOptions
// spliced in so a test can make exactly one field invalid.
func fileTargetCfg(t *testing.T, formatOptions string) string {
	t.Helper()

	opts := ""
	if formatOptions != "" {
		opts = `,"format_options":` + formatOptions
	}
	return `{"keep":{"type":"file","options":{"filename":"` +
		filepath.ToSlash(filepath.Join(t.TempDir(), "existing.log")) +
		`","max_size":1},"format":"plain"` + opts + `,"levels":[{"id":4,"name":"info"}]}}`
}

func TestConfigureTargetsRejectsUnsafeOptions(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "test.log")
	fileTarget := `"type":"file","options":{"filename":"` + logFile + `","max_size":1}`
	levels := `"levels":[{"id":4,"name":"info"}]`

	t.Run("baseline config is accepted", func(t *testing.T) {
		err := configureFromJSON(t, `{"t":{`+fileTarget+`,"format":"plain","format_options":{"delim":" | "},`+levels+`}}`)
		require.NoError(t, err)
	})

	t.Run("line_end payload is rejected", func(t *testing.T) {
		err := configureFromJSON(t, `{"t":{`+fileTarget+`,"format":"plain","format_options":{"line_end":"#!/bin/sh\nid\n"},`+levels+`}}`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "line_end")
	})

	t.Run("delim with newline is rejected", func(t *testing.T) {
		err := configureFromJSON(t, `{"t":{`+fileTarget+`,"format":"plain","format_options":{"delim":"\nforged "},`+levels+`}}`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "delim")
	})

	t.Run("oversized min_level_len is rejected", func(t *testing.T) {
		err := configureFromJSON(t, `{"t":{`+fileTarget+`,"format":"plain","format_options":{"min_level_len":2000000000},`+levels+`}}`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "min_level_len")
	})

	t.Run("level name payload is rejected", func(t *testing.T) {
		err := configureFromJSON(t, `{"t":{`+fileTarget+`,"format":"plain","levels":[{"id":4,"name":"info\npayload"}]}}`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "level name")
	})
}

// TestConfigureTargetsLeavesExistingTargets pins that a config which cannot be
// applied does not tear down the targets that are already working, whether it
// fails validation or fails to initialize.
func TestConfigureTargetsLeavesExistingTargets(t *testing.T) {
	initFailFactories := &Factories{
		TargetFactory: func(targetType string, options json.RawMessage) (logr.Target, error) {
			return &initFailTarget{}, nil
		},
	}

	cases := []struct {
		name      string
		bad       func(t *testing.T) string
		factories *Factories
	}{
		{
			name: "rejected by option validation",
			bad: func(t *testing.T) string {
				return fileTargetCfg(t, `{"line_end":"#!/bin/sh\nid\n"}`)
			},
		},
		{
			name: "target fails to initialize",
			bad: func(t *testing.T) string {
				return `{"broken":{"type":"custom","format":"plain","levels":[{"id":4,"name":"info"}]}}`
			},
			factories: initFailFactories,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lgr := newLgr(t)

			require.NoError(t, ConfigureTargets(lgr, parseCfg(t, fileTargetCfg(t, "")), nil))
			require.Len(t, lgr.TargetInfos(), 1, "the good config should be applied")

			require.Error(t, ConfigureTargets(lgr, parseCfg(t, tc.bad(t)), tc.factories))
			require.Len(t, lgr.TargetInfos(), 1,
				"the working target should survive a config that cannot be applied")
		})
	}
}

// initFailTarget is a target whose Init fails, as a syslog target does when the
// daemon is unreachable.
type initFailTarget struct{}

func (t *initFailTarget) Init() error { return errors.New("simulated init failure") }

func (t *initFailTarget) Write(p []byte, rec *logr.LogRec) (int, error) { return len(p), nil }

func (t *initFailTarget) Shutdown() error { return nil }

// validatedFormatter is a formatter from a factory that opts in to validation.
// logr.DefaultFormatter has no CheckValid, so it serves as the formatter that
// cannot validate itself.
type validatedFormatter struct {
	logr.DefaultFormatter
	lineEnd string
}

func (f *validatedFormatter) CheckValid() error {
	return logr.CheckOptionLineEnd("line_end", f.lineEnd)
}

func TestConfigureTargetsValidatesFactoryResults(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "factory.log")
	raw := `{"t":{"type":"file","options":{"filename":"` + logFile + `","max_size":1},` +
		`"format":"custom","levels":[{"id":4,"name":"info"}]}}`

	var cfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))

	newLgr := func(t *testing.T) *logr.Logr {
		t.Helper()
		lgr, err := logr.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = lgr.Shutdown()
		})
		return lgr
	}

	t.Run("rejects an invalid factory formatter", func(t *testing.T) {
		factories := &Factories{
			FormatterFactory: func(format string, options json.RawMessage) (logr.Formatter, error) {
				return &validatedFormatter{lineEnd: "#!/bin/sh\nid\n"}, nil
			},
		}
		err := ConfigureTargets(newLgr(t), cfg, factories)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid custom formatter options")
	})

	t.Run("accepts a valid factory formatter", func(t *testing.T) {
		factories := &Factories{
			FormatterFactory: func(format string, options json.RawMessage) (logr.Formatter, error) {
				return &validatedFormatter{lineEnd: "\n"}, nil
			},
		}
		require.NoError(t, ConfigureTargets(newLgr(t), cfg, factories))
	})

	t.Run("accepts a factory formatter that cannot validate itself", func(t *testing.T) {
		factories := &Factories{
			FormatterFactory: func(format string, options json.RawMessage) (logr.Formatter, error) {
				return &logr.DefaultFormatter{}, nil
			},
		}
		require.NoError(t, ConfigureTargets(newLgr(t), cfg, factories))
	})
}
