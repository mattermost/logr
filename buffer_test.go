package logr_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/require"
)

func TestBuffer_NewBuffer(t *testing.T) {
	buf := bytes.Buffer{}

	logBufferWriter := logr.NewBuffer(&buf)
	target := targets.NewWriterTarget(logBufferWriter)
	filter := &logr.CustomFilter{}
	filter.Add(logr.Debug)

	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "New Buffer Target", filter, &formatters.JSON{}, 1000)
	require.NoError(t, err)

	logger := lgr.NewLogger()

	for i := 1; i <= 5; i++ {
		logger.Debug("Debug message", logr.Any("count", i))
	}

	err = lgr.Shutdown()
	require.NoError(t, err)

	decoder := json.NewDecoder(
		bytes.NewBufferString(
			strings.TrimSpace(
				logBufferWriter.String(),
			),
		),
	)

	type logLine struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Message   string `json:"msg"`
		Count     int    `json:"count"`
	}

	countOfLogLine := 1
	for decoder.More() {
		line := logLine{}

		if err := decoder.Decode(&line); err != nil {
			t.Errorf("error while decoding json log line - err: %s", err.Error())
		}

		if countOfLogLine != line.Count {
			t.Errorf("log line mismatch - got value of count '%d', want '%d'", line.Count, countOfLogLine)
		}

		countOfLogLine++
	}
}
