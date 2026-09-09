package config

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/require"
)

// configureFromJSON builds a single-target config from raw JSON and applies it.
func configureFromJSON(t *testing.T, raw string) error {
	t.Helper()

	var cfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(raw), &cfg))

	lgr, err := logr.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = lgr.Shutdown()
	})

	return ConfigureTargets(lgr, cfg, nil)
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

// TestConfigureTargetsLeavesExistingTargetsOnError pins that a rejected config
// does not tear down the targets that are already working.
func TestConfigureTargetsLeavesExistingTargetsOnError(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "good.log")
	good := `{"t":{"type":"file","options":{"filename":"` + logFile + `","max_size":1},` +
		`"format":"plain","levels":[{"id":4,"name":"info"}]}}`

	var goodCfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(good), &goodCfg))

	lgr, err := logr.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = lgr.Shutdown()
	})

	require.NoError(t, ConfigureTargets(lgr, goodCfg, nil))
	require.Len(t, lgr.TargetInfos(), 1, "the good config should be applied")

	bad := `{"t":{"type":"file","options":{"filename":"` + logFile + `","max_size":1},` +
		`"format":"plain","format_options":{"line_end":"#!/bin/sh\nid\n"},"levels":[{"id":4,"name":"info"}]}}`

	var badCfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(bad), &badCfg))

	require.Error(t, ConfigureTargets(lgr, badCfg, nil))
	require.Len(t, lgr.TargetInfos(), 1, "the rejected config should leave the existing target in place")
}

// unvalidatedFormatter is a formatter from a factory that does not implement
// logr.Validator, so it is applied without any option limits.
type unvalidatedFormatter struct{}

func (f *unvalidatedFormatter) IsStacktraceNeeded() bool { return false }

func (f *unvalidatedFormatter) Format(rec *logr.LogRec, level logr.Level, buf *bytes.Buffer) (*bytes.Buffer, error) {
	if buf == nil {
		buf = &bytes.Buffer{}
	}
	buf.WriteString(rec.Msg())
	return buf, nil
}

// validatedFormatter is a formatter from a factory that opts in to validation.
type validatedFormatter struct {
	unvalidatedFormatter
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
		require.Contains(t, err.Error(), "invalid options from formatter factory")
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
				return &unvalidatedFormatter{}, nil
			},
		}
		require.NoError(t, ConfigureTargets(newLgr(t), cfg, factories))
	})
}
