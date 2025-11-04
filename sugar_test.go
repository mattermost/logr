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

// TestSugar_AllLogLevels tests all sugar logging levels
func TestSugar_AllLogLevels(t *testing.T) {
	// Use Trace level to capture all logs
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Trace}
	buf := &bytes.Buffer{}
	target := targets.NewWriterTarget(buf)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "sugarTest", filter, formatter, 3000)
	require.NoError(t, err)

	sugar := lgr.NewLogger().Sugar()

	// Test Trace
	sugar.Trace("trace message", "arg1")
	// Test Warn
	sugar.Warn("warn message", "arg2")
	// Test Print (should log at Info level)
	sugar.Print("print message", "arg3")

	err = lgr.Shutdown()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "trace message")
	assert.Contains(t, output, "arg1")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "arg2")
	assert.Contains(t, output, "print message")
	assert.Contains(t, output, "arg3")
}

// TestSugar_PrintfStyleMethods tests all Printf-style sugar methods
func TestSugar_PrintfStyleMethods(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Trace}
	buf := &bytes.Buffer{}
	target := targets.NewWriterTarget(buf)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "sugarTest", filter, formatter, 3000)
	require.NoError(t, err)

	sugar := lgr.NewLogger().Sugar()

	sugar.Tracef("trace %s", "formatted")
	sugar.Debugf("debug %s", "formatted")
	sugar.Infof("info %s", "formatted")
	sugar.Printf("printf %s", "formatted")
	sugar.Warnf("warn %s", "formatted")
	sugar.Errorf("error %s", "formatted")

	err = lgr.Shutdown()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "trace formatted")
	assert.Contains(t, output, "debug formatted")
	assert.Contains(t, output, "info formatted")
	assert.Contains(t, output, "printf formatted")
	assert.Contains(t, output, "warn formatted")
	assert.Contains(t, output, "error formatted")
}

// TestSugar_StructuredMethods tests all structured (w) sugar methods
func TestSugar_StructuredMethods(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Trace}
	buf := &bytes.Buffer{}
	target := targets.NewWriterTarget(buf)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "sugarTest", filter, formatter, 3000)
	require.NoError(t, err)

	sugar := lgr.NewLogger().Sugar()

	sugar.Tracew("trace msg", "key1", "val1")
	sugar.Infow("info msg", "key2", "val2")
	sugar.Warnw("warn msg", "key3", "val3")
	sugar.Errorw("error msg", "key4", "val4")

	err = lgr.Shutdown()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "trace msg")
	assert.Contains(t, output, "key1=val1")
	assert.Contains(t, output, "info msg")
	assert.Contains(t, output, "key2=val2")
	assert.Contains(t, output, "warn msg")
	assert.Contains(t, output, "key3=val3")
	assert.Contains(t, output, "error msg")
	assert.Contains(t, output, "key4=val4")
}

// TestSugar_LogfWithoutFormat tests Logf with empty format string
func TestSugar_LogfWithoutFormat(t *testing.T) {
	formatter := &formatters.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info}
	buf := &bytes.Buffer{}
	target := targets.NewWriterTarget(buf)
	lgr, _ := logr.New()
	err := lgr.AddTarget(target, "sugarTest", filter, formatter, 3000)
	require.NoError(t, err)

	sugar := lgr.NewLogger().Sugar()

	// Empty format string should use Sprint instead of Sprintf
	sugar.Logf(logr.Info, "", "arg1", "arg2", "arg3")

	err = lgr.Shutdown()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "arg1")
	assert.Contains(t, output, "arg2")
	assert.Contains(t, output, "arg3")
}
