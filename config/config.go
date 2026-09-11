// Package config provides utilities for configuring logr from JSON configuration files.
package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
)

// TargetCfg defines the configuration for a single log target, including
// its type, format, and output levels.
type TargetCfg struct {
	Type          string          `json:"type"` // one of "console", "file", "tcp", "syslog", "none".
	Options       json.RawMessage `json:"options,omitempty"`
	Format        string          `json:"format"` // one of "json", "plain", "gelf"
	FormatOptions json.RawMessage `json:"format_options,omitempty"`
	Levels        []logr.Level    `json:"levels"`
	MaxQueueSize  int             `json:"maxqueuesize,omitempty"`
}

// ConsoleOptions specifies options for console-based log targets.
type ConsoleOptions struct {
	Out string `json:"out"` // one of "stdout", "stderr"
}

// TargetFactory creates a log target from a type string and options.
type TargetFactory func(targetType string, options json.RawMessage) (logr.Target, error)

// FormatterFactory creates a log formatter from a format string and options.
type FormatterFactory func(format string, options json.RawMessage) (logr.Formatter, error)

// Factories provides custom factories for creating targets and formatters
// that are not built-in to logr.
type Factories struct {
	TargetFactory    TargetFactory    // can be nil
	FormatterFactory FormatterFactory // can be nil
}

// ConfigureTargets replaces the current list of log targets with a new one based on a map
// of name->TargetCfg. The map of TargetCfg's would typically be serialized from a JSON
// source or can be programmatically created.
//
// An optional set of factories can be provided which will be called to create any target
// types or formatters not built-in.
//
// Every target, formatter and filter in the config is created, validated and
// initialized before any existing target is removed, so a config that cannot be
// applied leaves the current targets untouched.
//
// To append a log target to an existing config, use `(*Logr).AddTarget` instead.
func ConfigureTargets(lgr *logr.Logr, config map[string]TargetCfg, factories *Factories) error {
	if factories == nil {
		factories = &Factories{nil, nil}
	}

	specs := make([]logr.TargetSpec, 0, len(config))

	for name, tcfg := range config {
		target, err := newTarget(tcfg.Type, tcfg.Options, factories.TargetFactory)
		if err != nil {
			return fmt.Errorf("error creating log target %s: %w", name, err)
		}

		if target == nil {
			continue
		}

		formatter, err := newFormatter(tcfg.Format, tcfg.FormatOptions, factories.FormatterFactory)
		if err != nil {
			return fmt.Errorf("error creating formatter for log target %s: %w", name, err)
		}

		filter, err := newFilter(tcfg.Levels)
		if err != nil {
			return fmt.Errorf("error creating filter for log target %s: %w", name, err)
		}

		specs = append(specs, logr.TargetSpec{
			Target:       target,
			Name:         name,
			Filter:       filter,
			Formatter:    formatter,
			MaxQueueSize: tcfg.MaxQueueSize,
		})
	}

	return lgr.ReplaceTargets(context.Background(), specs)
}

// checkValid validates v if it can validate itself. `logr.Target` and
// `logr.Formatter` do not declare CheckValid, so a factory-supplied
// implementation is held to the option limits only if it opts in.
func checkValid(v any) error {
	if val, ok := v.(logr.Validator); ok {
		return val.CheckValid()
	}
	return nil
}

func newFilter(levels []logr.Level) (logr.Filter, error) {
	filter := &logr.CustomFilter{}
	for _, lvl := range levels {
		if err := lvl.CheckValid(); err != nil {
			return nil, err
		}
		filter.Add(lvl)
	}
	return filter, nil
}

func newTarget(targetType string, options json.RawMessage, factory TargetFactory) (logr.Target, error) {
	switch strings.ToLower(targetType) {
	case "console":
		c := ConsoleOptions{}
		if len(options) != 0 {
			if err := json.Unmarshal(options, &c); err != nil {
				return nil, fmt.Errorf("error decoding console target options: %w", err)
			}
		}
		var w io.Writer
		switch c.Out {
		case "stderr":
			w = os.Stderr
		case "stdout", "":
			w = os.Stdout
		default:
			return nil, fmt.Errorf("invalid console target option '%s'", c.Out)
		}
		return targets.NewWriterTarget(w), nil
	case "file":
		fo := targets.FileOptions{}
		if len(options) == 0 {
			return nil, errors.New("missing file target options")
		}
		if err := json.Unmarshal(options, &fo); err != nil {
			return nil, fmt.Errorf("error decoding file target options: %w", err)
		}
		if err := fo.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid file target options: %w", err)
		}
		return targets.NewFileTarget(fo), nil
	case "tcp":
		to := targets.TcpOptions{}
		if len(options) == 0 {
			return nil, errors.New("missing TCP target options")
		}
		if err := json.Unmarshal(options, &to); err != nil {
			return nil, fmt.Errorf("error decoding TCP target options: %w", err)
		}
		if err := to.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid TCP target options: %w", err)
		}
		return targets.NewTcpTarget(&to), nil
	case "syslog":
		so := targets.SyslogOptions{}
		if len(options) == 0 {
			return nil, errors.New("missing SysLog target options")
		}
		if err := json.Unmarshal(options, &so); err != nil {
			return nil, fmt.Errorf("error decoding Syslog target options: %w", err)
		}
		if err := so.CheckValid(); err != nil {
			return nil, fmt.Errorf("invalid SysLog target options: %w", err)
		}
		return targets.NewSyslogTarget(&so)
	case "none":
		return nil, nil
	default:
		if factory != nil {
			t, err := factory(targetType, options)
			if err != nil || t == nil {
				return nil, fmt.Errorf("error from target factory: %w", err)
			}
			if err := checkValid(t); err != nil {
				return nil, fmt.Errorf("invalid options from target factory: %w", err)
			}
			return t, nil
		}
	}
	return nil, fmt.Errorf("target type '%s' is unrecognized", targetType)
}

func newFormatter(format string, options json.RawMessage, factory FormatterFactory) (logr.Formatter, error) {
	formatter, err := buildFormatter(format, options, factory)
	if err != nil {
		return nil, err
	}
	if err := checkValid(formatter); err != nil {
		return nil, fmt.Errorf("invalid %s formatter options: %w", format, err)
	}
	return formatter, nil
}

func buildFormatter(format string, options json.RawMessage, factory FormatterFactory) (logr.Formatter, error) {
	switch strings.ToLower(format) {
	case "json":
		j := &formatters.JSON{}
		if err := decodeOptions(options, j); err != nil {
			return nil, fmt.Errorf("error decoding JSON formatter options: %w", err)
		}
		return j, nil
	case "plain":
		p := &formatters.Plain{}
		if err := decodeOptions(options, p); err != nil {
			return nil, fmt.Errorf("error decoding Plain formatter options: %w", err)
		}
		return p, nil
	case "gelf":
		g := &formatters.Gelf{}
		if err := decodeOptions(options, g); err != nil {
			return nil, fmt.Errorf("error decoding Gelf formatter options: %w", err)
		}
		return g, nil
	default:
		if factory != nil {
			f, err := factory(format, options)
			if err != nil || f == nil {
				return nil, fmt.Errorf("error from formatter factory: %w", err)
			}
			return f, nil
		}
	}
	return nil, fmt.Errorf("format '%s' is unrecognized", format)
}

// decodeOptions unmarshals options into target, treating an empty value as
// "keep the defaults".
func decodeOptions(options json.RawMessage, target any) error {
	if len(options) == 0 {
		return nil
	}
	return json.Unmarshal(options, target)
}
