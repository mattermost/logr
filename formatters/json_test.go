package formatters_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	formatter := &formatters.JSON{DisableTimestamp: true, DisableStacktrace: true}

	t.Run("default sorter, one field", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger().WithField("name", "wiggin")

		logger.Error("This is an error.")
		lgr.Flush()

		want := NL(`{"level":"error","msg":"This is an error.","name":"wiggin"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("default sorter, zero fields", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger()

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error."}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("default sorter, three fields", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		fields := logr.Fields{}
		fields["middle_name"] = "Thomas"
		fields["last_name"] = "Wiggin"
		fields["first_name"] = "Ender"
		logger := lgr.NewLogger().WithFields(fields)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","first_name":"Ender","last_name":"Wiggin","middle_name":"Thomas"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("default sorter, three fields, context grouped", func(t *testing.T) {
		formatter := &formatters.JSON{DisableTimestamp: true, DisableStacktrace: true, KeyContextFields: "ctx"}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		fields := logr.Fields{}
		fields["middle_name"] = "Thomas"
		fields["last_name"] = "Wiggin"
		fields["first_name"] = "Ender"
		logger := lgr.NewLogger().WithFields(fields)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","ctx":{"first_name":"Ender","last_name":"Wiggin","middle_name":"Thomas"}}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("reverse sorter, three fields", func(t *testing.T) {
		formatterWithReverseSort := &formatters.JSON{DisableTimestamp: true, DisableStacktrace: true, ContextSorter: reverseSort}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatterWithReverseSort, 1000)
		require.NoError(t, err)

		fields := logr.Fields{}
		fields["last_name"] = "Wiggin"
		fields["middle_name"] = "Thomas"
		fields["first_name"] = "Ender"
		logger := lgr.NewLogger().WithFields(fields)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","middle_name":"Thomas","last_name":"Wiggin","first_name":"Ender"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("reverse sorter, three fields, context grouped", func(t *testing.T) {
		formatter := &formatters.JSON{DisableTimestamp: true, DisableStacktrace: true, ContextSorter: reverseSort, KeyContextFields: "ctx"}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		fields := logr.Fields{}
		fields["last_name"] = "Wiggin"
		fields["middle_name"] = "Thomas"
		fields["first_name"] = "Ender"
		logger := lgr.NewLogger().WithFields(fields)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","ctx":{"middle_name":"Thomas","last_name":"Wiggin","first_name":"Ender"}}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	err := lgr.Shutdown()
	require.NoError(t, err)
}

func reverseSort(fields logr.Fields) []formatters.ContextField {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	cf := make([]formatters.ContextField, 0, len(keys))
	for _, k := range keys {
		cf = append(cf, formatters.ContextField{Key: k, Val: fields[k]})
	}
	return cf
}

func NL(s string) string {
	return s + "\n"
}
