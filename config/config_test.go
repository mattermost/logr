package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	Server   = "localhost"
	TestPort = 18066
)

func TestConfigureTargets(t *testing.T) {
	b, err := os.ReadFile("sample-config.json")
	require.NoError(t, err, "should read file without error")

	var cfg map[string]TargetCfg
	err = json.Unmarshal(b, &cfg)
	require.NoError(t, err, "should unmarshall without error")

	buf := &test.Buffer{}
	server, err := test.NewSocketServer(TestPort, buf)
	require.NoError(t, err)

	// Add error handler to capture any connection errors
	opt := logr.OnLoggerError(func(err error) {
		t.Logf("Logger error: %v", err)
	})
	lgr, err := logr.New(opt)
	require.NoError(t, err)

	// Small delay to ensure server is listening
	time.Sleep(100 * time.Millisecond)

	err = ConfigureTargets(lgr, cfg, nil)
	require.NoError(t, err)

	// ConfigureTargets will cause Syslog.Init() to connect synchronously
	// Wait for that connection to be registered
	err = server.WaitForAnyConnection()
	require.NoError(t, err)

	logger := lgr.NewLogger().With(logr.String("test", "echo"))

	logger.Debug("Unique sum")

	// Flush to ensure logs are sent
	err = lgr.Flush()
	require.NoError(t, err)

	err = lgr.Shutdown()
	require.NoError(t, err)

	// Give time for connections to close after shutdown
	time.Sleep(100 * time.Millisecond)

	err = server.StopServer(false)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Unique sum")
	assert.Contains(t, buf.String(), "echo")
}

func TestConfigureCustomTarget(t *testing.T) {
	str := `{    "sample-custom": {
        "type": "my_custom_target",
        "options": {
            "custom_prop": "hello"
        },
        "format": "my_custom_format",
        "format_options": {
            "foo": "bar"
        },
        "levels": [
            {"id": 5, "name": "debug"}
        ],
        "maxqueuesize": 1000
    } }`

	var cfg map[string]TargetCfg
	err := json.Unmarshal([]byte(str), &cfg)
	require.NoError(t, err, "should unmarshall without error")

	buf := &test.Buffer{}

	factories := Factories{
		TargetFactory:    makeCustomTargetFactory(buf),
		FormatterFactory: customFormatFactory,
	}

	lgr, err := logr.New()
	require.NoError(t, err)

	err = ConfigureTargets(lgr, cfg, &factories)
	require.NoError(t, err)

	logger := lgr.NewLogger().With(logr.String("test", "mode"))

	logger.Debug("Unique foo")

	err = lgr.Shutdown()
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Unique foo")
	assert.Contains(t, buf.String(), "mode")
}

func makeCustomTargetFactory(w io.Writer) TargetFactory {
	return func(targetType string, options json.RawMessage) (logr.Target, error) {
		if targetType != "my_custom_target" {
			return nil, fmt.Errorf("unknown type %s", targetType)
		}
		return targets.NewWriterTarget(w), nil
	}
}

func customFormatFactory(format string, options json.RawMessage) (logr.Formatter, error) {
	if format != "my_custom_format" {
		return nil, fmt.Errorf("unknown format %s", format)
	}
	return &formatters.Plain{Delim: " / "}, nil
}

func TestConfigureTargetsTruncatesLevelName(t *testing.T) {
	name := strings.Repeat("L", logr.MaxLevelNameLen+1)
	str := fmt.Sprintf(`{"long-level": {
        "type": "my_custom_target",
        "format": "plain",
        "levels": [{"id": 5, "name": %q}]
    }}`, name)

	var cfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(str), &cfg))

	orig := logr.TruncateWriter
	t.Cleanup(func() { logr.TruncateWriter = orig })
	warn := &bytes.Buffer{}
	logr.TruncateWriter = warn

	buf := &test.Buffer{}
	lgr, err := logr.New()
	require.NoError(t, err)
	require.NoError(t, ConfigureTargets(lgr, cfg, &Factories{TargetFactory: makeCustomTargetFactory(buf)}))

	lgr.NewLogger().Log(logr.Level{ID: 5}, "hello")
	require.NoError(t, lgr.Shutdown())

	assert.True(t, strings.HasPrefix(buf.String(), name[:logr.MaxLevelNameLen]+" "), buf.String())
	assert.Equal(t, 1, strings.Count(warn.String(), "level name is too long"))
}

func TestConfigureTargetsRejectsLevelNameControlCharacters(t *testing.T) {
	str := `{"bad-level": {
        "type": "my_custom_target",
        "format": "plain",
        "levels": [{"id": 5, "name": "\n2026-10-24 fake [x] admin login ok\n"}]
    }}`

	var cfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(str), &cfg))

	lgr, err := logr.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = lgr.Shutdown() })

	err = ConfigureTargets(lgr, cfg, &Factories{TargetFactory: makeCustomTargetFactory(&test.Buffer{})})
	require.Error(t, err)
	require.Contains(t, err.Error(), "level name contains a control character")
}

func TestConfigureTargetsRejectsLevelNameLineSeparator(t *testing.T) {
	str := `{"bad-level": {
        "type": "my_custom_target",
        "format": "plain",
        "levels": [{"id": 5, "name": "info 2026-10-24 fake [x] admin login ok"}]
    }}`

	var cfg map[string]TargetCfg
	require.NoError(t, json.Unmarshal([]byte(str), &cfg))

	lgr, err := logr.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = lgr.Shutdown() })

	err = ConfigureTargets(lgr, cfg, &Factories{TargetFactory: makeCustomTargetFactory(&test.Buffer{})})
	require.Error(t, err)
	require.Contains(t, err.Error(), "level name contains a line separator")
}
