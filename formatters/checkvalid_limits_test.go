package formatters_test

import (
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/stretchr/testify/require"
)

func TestPlainCheckValidLimits(t *testing.T) {
	t.Run("defaults are valid", func(t *testing.T) {
		require.NoError(t, (&formatters.Plain{}).CheckValid())
	})

	t.Run("typical options are valid", func(t *testing.T) {
		p := &formatters.Plain{
			Delim:           " | ",
			LineEnd:         "\r\n",
			MinLevelLen:     5,
			MinMessageLen:   40,
			TimestampFormat: logr.DefTimestampFormat,
		}
		require.NoError(t, p.CheckValid())
	})

	t.Run("line_end cannot carry a payload", func(t *testing.T) {
		p := &formatters.Plain{LineEnd: "\n#!/bin/sh\nid > /tmp/pwned\n"}
		require.Error(t, p.CheckValid())
	})

	t.Run("line_end is length limited", func(t *testing.T) {
		p := &formatters.Plain{LineEnd: strings.Repeat("\n", logr.MaxLineEndLen+1)}
		require.Error(t, p.CheckValid())
	})

	t.Run("delim cannot forge log lines", func(t *testing.T) {
		p := &formatters.Plain{Delim: "\nforged "}
		require.Error(t, p.CheckValid())
	})

	t.Run("tab delimiter is allowed", func(t *testing.T) {
		require.NoError(t, (&formatters.Plain{Delim: "\t"}).CheckValid())
	})

	t.Run("line_end must contain a line break", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{LineEnd: "   "}).CheckValid())
	})

	t.Run("empty line_end means the default", func(t *testing.T) {
		require.NoError(t, (&formatters.Plain{LineEnd: ""}).CheckValid())
	})

	t.Run("delim is length limited", func(t *testing.T) {
		p := &formatters.Plain{Delim: strings.Repeat("x", logr.MaxDelimLen+1)}
		require.Error(t, p.CheckValid())
	})

	t.Run("min_level_len is bounded", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{MinLevelLen: -1}).CheckValid())
		require.Error(t, (&formatters.Plain{MinLevelLen: logr.MaxPadLen + 1}).CheckValid())
	})

	t.Run("min_msg_len is bounded", func(t *testing.T) {
		require.Error(t, (&formatters.Plain{MinMessageLen: -1}).CheckValid())
		require.Error(t, (&formatters.Plain{MinMessageLen: logr.MaxPadLen + 1}).CheckValid())
	})

	t.Run("timestamp_format is length limited", func(t *testing.T) {
		p := &formatters.Plain{TimestampFormat: strings.Repeat("2006", logr.MaxTimestampFormatLen)}
		require.Error(t, p.CheckValid())
	})
}

func TestJSONCheckValidLimits(t *testing.T) {
	t.Run("defaults are valid", func(t *testing.T) {
		require.NoError(t, (&formatters.JSON{}).CheckValid())
	})

	t.Run("custom keys are valid", func(t *testing.T) {
		j := &formatters.JSON{KeyTimestamp: "ts", KeyLevel: "lvl", KeyMsg: "message"}
		require.NoError(t, j.CheckValid())
	})

	t.Run("keys are length limited", func(t *testing.T) {
		j := &formatters.JSON{KeyMsg: strings.Repeat("k", logr.MaxFieldKeyLen+1)}
		require.Error(t, j.CheckValid())
	})

	t.Run("keys reject control characters", func(t *testing.T) {
		require.Error(t, (&formatters.JSON{KeyCaller: "call\ner"}).CheckValid())
	})

	t.Run("timestamp_format is length limited", func(t *testing.T) {
		j := &formatters.JSON{TimestampFormat: strings.Repeat("2006", logr.MaxTimestampFormatLen)}
		require.Error(t, j.CheckValid())
	})
}
