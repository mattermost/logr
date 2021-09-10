package logr_test

import (
	"bytes"
	"strings"
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

type kv []interface{}

const (
	pre           = "debug | test msg | test=sugar "
	errInvalidKey = "error | invalid key"
)

func TestSugar_argsToFields(t *testing.T) {
	tests := []struct {
		name          string
		keyValuePairs kv
		want          string
		wantErr       bool
	}{
		{name: "one pair", keyValuePairs: kv{"prop1", 7}, want: pre + "prop1=7"},
		{name: "two pair", keyValuePairs: kv{"prop1", 11, "prop2", "bar"}, want: pre + "prop1=11 prop2=bar"},
		{name: "empty", keyValuePairs: kv{}, want: pre},
		{name: "invalid key", keyValuePairs: kv{200, 300}, wantErr: true},
		{name: "one arg", keyValuePairs: kv{"bad"}, wantErr: true},
		{name: "one arg invalid", keyValuePairs: kv{22}, wantErr: true},
		{name: "nil args", keyValuePairs: nil, want: pre},
		{name: "dangling key", keyValuePairs: kv{"prop1", 7, "dangle"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			sugar, shutdown, err := makeSugar(buf)
			require.NoError(t, err)

			sugar.Debugw("test msg", tt.keyValuePairs...)

			err = shutdown()
			require.NoError(t, err)

			got := strings.TrimSpace(buf.String())
			want := strings.TrimSpace(tt.want)

			if tt.wantErr {
				assert.Contains(t, got, errInvalidKey)
			} else {
				assert.Equal(t, want, got)
			}
		})
	}
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
