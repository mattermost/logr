package logr_test

import (
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/assert"
)

func TestStdFilter_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		filter   logr.StdFilter
		level    logr.Level
		expected bool
	}{
		{"Panic at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Panic, true},
		{"Fatal at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Fatal, true},
		{"Error at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Error, true},
		{"Warn at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Warn, false},
		{"Info at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Info, false},
		{"Debug at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Debug, false},
		{"Trace at Error level", logr.StdFilter{Lvl: logr.Error}, logr.Trace, false},

		{"Panic at Info level", logr.StdFilter{Lvl: logr.Info}, logr.Panic, true},
		{"Error at Info level", logr.StdFilter{Lvl: logr.Info}, logr.Error, true},
		{"Warn at Info level", logr.StdFilter{Lvl: logr.Info}, logr.Warn, true},
		{"Info at Info level", logr.StdFilter{Lvl: logr.Info}, logr.Info, true},
		{"Debug at Info level", logr.StdFilter{Lvl: logr.Info}, logr.Debug, false},
		{"Trace at Info level", logr.StdFilter{Lvl: logr.Info}, logr.Trace, false},

		{"All at Trace level", logr.StdFilter{Lvl: logr.Trace}, logr.Trace, true},
		{"All at Trace level 2", logr.StdFilter{Lvl: logr.Trace}, logr.Debug, true},
		{"All at Trace level 3", logr.StdFilter{Lvl: logr.Trace}, logr.Panic, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsEnabled(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStdFilter_IsStacktraceEnabled(t *testing.T) {
	tests := []struct {
		name     string
		filter   logr.StdFilter
		level    logr.Level
		expected bool
	}{
		{"Panic with Error stacktrace", logr.StdFilter{Stacktrace: logr.Error}, logr.Panic, true},
		{"Fatal with Error stacktrace", logr.StdFilter{Stacktrace: logr.Error}, logr.Fatal, true},
		{"Error with Error stacktrace", logr.StdFilter{Stacktrace: logr.Error}, logr.Error, true},
		{"Warn with Error stacktrace", logr.StdFilter{Stacktrace: logr.Error}, logr.Warn, false},
		{"Info with Error stacktrace", logr.StdFilter{Stacktrace: logr.Error}, logr.Info, false},

		{"Panic with Panic stacktrace", logr.StdFilter{Stacktrace: logr.Panic}, logr.Panic, true},
		{"Fatal with Panic stacktrace", logr.StdFilter{Stacktrace: logr.Panic}, logr.Fatal, false},
		{"Error with Panic stacktrace", logr.StdFilter{Stacktrace: logr.Panic}, logr.Error, false},

		{"All with Trace stacktrace", logr.StdFilter{Stacktrace: logr.Trace}, logr.Trace, true},
		{"All with Trace stacktrace 2", logr.StdFilter{Stacktrace: logr.Trace}, logr.Info, true},
		{"All with Trace stacktrace 3", logr.StdFilter{Stacktrace: logr.Trace}, logr.Panic, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsStacktraceEnabled(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStdFilter_GetEnabledLevel(t *testing.T) {
	tests := []struct {
		name              string
		filter            logr.StdFilter
		level             logr.Level
		expectedEnabled   bool
		expectedStacktrace bool
	}{
		{"Error enabled with Error filter", logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}, logr.Error, true, true},
		{"Error enabled, no stacktrace", logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Panic}, logr.Error, true, false},
		{"Info disabled at Error filter", logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}, logr.Info, false, false},
		{"Panic with stacktrace", logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}, logr.Panic, true, true},
		{"Debug enabled at Trace", logr.StdFilter{Lvl: logr.Trace, Stacktrace: logr.Panic}, logr.Debug, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, enabled := tt.filter.GetEnabledLevel(tt.level)
			assert.Equal(t, tt.expectedEnabled, enabled)
			if enabled {
				assert.Equal(t, tt.expectedStacktrace, level.Stacktrace)
			}
		})
	}
}
