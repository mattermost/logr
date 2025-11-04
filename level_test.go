package logr_test

import (
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    logr.Level
		expected string
	}{
		{logr.Panic, "panic"},
		{logr.Fatal, "fatal"},
		{logr.Error, "error"},
		{logr.Warn, "warn"},
		{logr.Info, "info"},
		{logr.Debug, "debug"},
		{logr.Trace, "trace"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestLevel_CustomLevel(t *testing.T) {
	customLevel := logr.Level{
		ID:   999,
		Name: "custom",
	}

	assert.Equal(t, "custom", customLevel.String())
}
