package logr_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSugarLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	sugar, shutdown, err := makeSugar(buf)
	require.NoError(t, err)

	// Info
	sugar.Info("Test for info level", "ident1", "ident2", 77)

	// Error with stacktrace
	sugar.Error("Test for error level", "ident3", "ident4", 33)

	// Debugw
	sugar.Debugw("Test for error level with name/value pairs", "prop1", "ident6", "prop2", "ident7")

	// Debugw no args
	sugar.Debugw("Test name/value pairs no args")

	// Debugw invalid args
	sugar.Debugw("Test name/value pairs invalid args1", 44, "hello")

	// With
	sugar2 := sugar.With("prop3", "foo", "prop4", "bar")
	sugar2.Debug("Test With")

	err = shutdown()
	require.NoError(t, err)
	data := buf.String()

	// Info
	assert.Contains(t, data, "test=sugar")
	assert.Contains(t, data, "Test for info level")
	assert.Contains(t, data, "ident1")
	assert.Contains(t, data, "ident2")
	assert.Contains(t, data, "=77")

	// Error
	assert.Contains(t, data, "test=sugar")
	assert.Contains(t, data, "Test for error level")
	assert.Contains(t, data, "ident3")
	assert.Contains(t, data, "ident4")
	assert.Contains(t, data, "=33")
	assert.Contains(t, data, "logr/sugar_test.go:")

	// Debugw
	assert.Contains(t, data, "test=sugar")
	assert.Contains(t, data, "Test for error level with name/value pairs")
	assert.Contains(t, data, "prop1=ident6")
	assert.Contains(t, data, "prop2=ident7")

	// Debugw no args
	assert.Contains(t, data, "test=sugar")
	assert.Contains(t, data, "Test name/value pairs no args")

	// invalid args
	assert.Contains(t, data, "invalid key for key/value pair")

	// With
	assert.Contains(t, data, "test=sugar")
	assert.Contains(t, data, "Test With")
	assert.Contains(t, data, "prop3=foo")
	assert.Contains(t, data, "prop4=bar")
}

func makeSugar(buf *bytes.Buffer) (logr.Sugar, func() error, error) {
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Debug, Stacktrace: logr.Error}
	target := targets.NewWriterTarget(buf)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "sugarTest", filter, formatter, 3000)
	if err != nil {
		return logr.Sugar{}, nil, err
	}
	sugar := lgr.NewLogger().Sugar(logr.String("test", "sugar"))
	shutdown := func() error {
		return lgr.Shutdown()
	}
	return sugar, shutdown, nil
}
