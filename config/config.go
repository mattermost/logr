package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
)

type TargetCfg struct {
	Type          string          `json:"type"`   // one of "console", "file", "tcp", "syslog", "none".
	Format        string          `json:"format"` // one of "json", "plain", "gelf"
	FormatOptions json.RawMessage `json:"format_options,omitempty"`
	Levels        []logr.Level    `json:"levels"`
	Options       json.RawMessage `json:"options,omitempty"`
	MaxQueueSize  int             `json:"maxqueuesize,omitempty"`
}

type TargetFactory func(targetType string, options json.RawMessage) (logr.Target, error)
type FormatterFactory func(format string, options json.RawMessage) (logr.Formatter, error)

type Factories struct {
	targetFactory    TargetFactory    // can be nil
	formatterFactory FormatterFactory // can be nil
}

var removeAll = func(ti logr.TargetInfo) bool { return true }

type errUnrecognizedFormat struct {
	format string
}

func (e *errUnrecognizedFormat) Error() string {
	return e.format + " is not recognized"
}

type errUnrecognizedTargetType struct {
	targetType string
}

func (e *errUnrecognizedTargetType) Error() string {
	return e.targetType + " is not recognized"
}

// ConfigureTargets replaces the current list of log targets with a new one based on a map
// of name->TargetCfg. The map of TargetCfg's would typically be serialized from a JSON
// source or can be programmatically created.
//
// An optional set of factories can be provided which will be called to create any target
// types or formatters not built-in.
//
// To append log targets to an existing config, use `(*Logr).AddTarget` or
// `(*Logr).AddTargetFromConfig` instead.
func ConfigureTargets(lgr *logr.Logr, config map[string]TargetCfg, factories *Factories) error {
	if err := lgr.RemoveTargets(context.Background(), removeAll); err != nil {
		return fmt.Errorf("error removing existing log targets: %w", err)
	}

	for name, tcfg := range config {
		target, err := newTarget(tcfg.Type, tcfg.Options, factories.targetFactory)
		if err != nil {
			return fmt.Errorf("error creating log target %s: %w", name, err)
		}

		if target == nil {
			continue
		}

		formatter, err := newFormatter(tcfg.Format, tcfg.FormatOptions, factories.formatterFactory)
		if err != nil {
			return fmt.Errorf("error creating formatter for log target %s: %w", name, err)
		}

		filter := newFilter(tcfg.Levels)
		qSize := tcfg.MaxQueueSize
		if qSize == 0 {
			qSize = logr.DefaultMaxQueueSize
		}

		if err = lgr.AddTarget(target, name, filter, formatter, qSize); err != nil {
			return fmt.Errorf("error adding log target %s: %w", name, err)
		}
	}
	return nil
}

func newFilter(levels []logr.Level) logr.Filter {
	filter := &logr.CustomFilter{}
	for _, lvl := range levels {
		filter.Add(lvl)
	}
	return filter
}

func newTarget(targetType string, options json.RawMessage, factory TargetFactory) (logr.Target, error) {
	switch strings.ToLower(targetType) {
	case "console":
	case "file":
	case "tcp":
	case "syslog":
	case "none":
		return nil, nil
	default:
		if factory != nil {
			t, err := factory(targetType, options)
			if err != nil || t == nil {
				return nil, fmt.Errorf("error from target factory: %w", err)
			}
		}
	}
	return nil, fmt.Errorf("target type '%s' is unrecogized", targetType)
}

func newFormatter(format string, options json.RawMessage, factory FormatterFactory) (logr.Formatter, error) {
	switch strings.ToLower(format) {
	case "json":
		j := formatters.JSON{}
		if len(options) != 0 {
			if err := json.Unmarshal(options, &j); err != nil {
				return nil, fmt.Errorf("error decoding JSON formatter options: %w", err)
			}
		}
		return &j, nil
	case "plain":
		p := formatters.Plain{}
		if len(options) != 0 {
			if err := json.Unmarshal(options, &p); err != nil {
				return nil, fmt.Errorf("error decoding Plain formatter options: %w", err)
			}
		}
		return &p, nil
	case "gelf":
		g := formatters.Gelf{}
		if len(options) != 0 {
			if err := json.Unmarshal(options, &g); err != nil {
				return nil, fmt.Errorf("error decoding Gelf formatter options: %w", err)
			}
		}
		return &g, nil

	default:
		if factory != nil {
			f, err := factory(format, options)
			if err != nil || f == nil {
				return nil, fmt.Errorf("error from formatter factory: %w", err)
			}
			return f, nil
		}
	}
	return nil, fmt.Errorf("format '%s' is unrecogized", format)
}
