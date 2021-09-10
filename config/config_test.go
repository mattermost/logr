package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

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
	b, err := ioutil.ReadFile("sample-config.json")
	require.NoError(t, err, "should read file without error")

	var cfg map[string]TargetCfg
	err = json.Unmarshal(b, &cfg)
	require.NoError(t, err, "should unmarshall without error")

	buf := &test.Buffer{}
	server, err := test.NewSocketServer(TestPort, buf)
	require.NoError(t, err)

	lgr, err := logr.New()
	require.NoError(t, err)

	err = ConfigureTargets(lgr, cfg, nil)
	require.NoError(t, err)

	logger := lgr.NewLogger().With(logr.String("test", "echo"))

	logger.Debug("Unique sum")

	err = lgr.Shutdown()
	require.NoError(t, err)

	err = server.WaitForAnyConnection()
	require.NoError(t, err)

	err = server.StopServer(true)
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
