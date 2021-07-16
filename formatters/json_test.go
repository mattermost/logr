package formatters_test

import (
	"errors"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Props struct {
	Key  string
	Blap int
}

type User struct {
	Name  string
	Age   int
	Props *Props
}

func TestJSONFieldTypes(t *testing.T) {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	formatter := &formatters.JSON{
		DisableTimestamp:  true,
		DisableStacktrace: true,
	}

	t.Run("basic types", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "basicTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger()

		logger.Error("Basic types test",
			logr.String("f1", "one"),
			logr.Int("f2", 77),
			logr.Bool("f3", true),
			logr.Float64("f4", 3.14),
			logr.Err(errors.New("test error")),
		)
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"Basic types test","f1":"one","f2":77,"f3":true,"f4":3.14,"error":"test error"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("time types", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "timeTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger()

		now, _ := time.Parse(logr.DefTimestampFormat, "2021-05-16 22:23:10.989 -04:00")
		millis := int64(1621218819966) // May 16, 2021 22:33:39.966
		dur := (time.Hour * 1) + (time.Minute * 34) + (time.Second * 17) + (time.Millisecond * 230)

		logger.Error("Time types test",
			logr.Time("f1", now),
			logr.Millis("f2", millis),
			logr.Duration("f3", dur),
		)
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"Time types test","f1":"2021-05-16 22:23:10.989 -04:00","f2":"May 17 02:33:39.966","f3":"1h34m17.23s"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("struct types", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "structTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger()

		user := User{Name: "wiggin", Age: 13, Props: &Props{Key: "foo", Blap: 77}}

		logger.Error("Struct types test",
			logr.Any("f1", user),
			logr.Any("f2", &user),
		)
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"Struct types test","f1":{"Name":"wiggin","Age":13,"Props":{"Key":"foo","Blap":77}},"f2":{"Name":"wiggin","Age":13,"Props":{"Key":"foo","Blap":77}}}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("array type", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "arrayTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger()

		f1 := []int{2, 4, 6, 8}
		f2 := []*User{
			{Name: "wiggin", Age: 13, Props: &Props{Key: "foo", Blap: 77}},
			{Name: "Jude", Age: 44, Props: &Props{Key: "foo", Blap: 78}},
		}

		logger.Error("Array test",
			logr.Array("f1", f1),
			logr.Array("f2", f2),
		)
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"Array test","f1":[2,4,6,8],"f2":[{"Name":"wiggin","Age":13,"Props":{"Key":"foo","Blap":77}},{"Name":"Jude","Age":44,"Props":{"Key":"foo","Blap":78}}]}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("map type", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "mapTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger()

		f1 := map[string]int{"two": 2, "four": 4, "six": 6, "eight": 8}
		f2 := map[string]*User{
			"one": {Name: "wiggin", Age: 13, Props: &Props{Key: "foo", Blap: 77}},
			"two": {Name: "Jude", Age: 44, Props: &Props{Key: "foo", Blap: 78}},
		}

		logger.Error("Array test",
			logr.Map("f1", f1),
			logr.Map("f2", f2),
		)
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"Array test","f1":{"eight":8,"four":4,"six":6,"two":2},"f2":{"one":{"Name":"wiggin","Age":13,"Props":{"Key":"foo","Blap":77}},"two":{"Name":"Jude","Age":44,"Props":{"Key":"foo","Blap":78}}}}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	err := lgr.Shutdown()
	require.NoError(t, err)
}

func TestJSON(t *testing.T) {
	lgr, _ := logr.New()
	filter := &logr.StdFilter{Lvl: logr.Error, Stacktrace: logr.Error}
	formatter := &formatters.JSON{
		DisableTimestamp:  true,
		DisableStacktrace: true,
		FieldSorter:       sorter,
	}

	t.Run("sorted, one field", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		if err != nil {
			t.Error(err)
		}

		logger := lgr.NewLogger().With(logr.String("name", "wiggin"))

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","name":"wiggin"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("sorted, zero fields", func(t *testing.T) {
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

	t.Run("sorted, three fields", func(t *testing.T) {
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger().With(
			logr.String("middle_name", "Thomas"),
			logr.String("last_name", "Wiggin"),
			logr.String("first_name", "Ender"),
		)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","first_name":"Ender","last_name":"Wiggin","middle_name":"Thomas"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("sorted, three fields, grouped", func(t *testing.T) {
		formatter := &formatters.JSON{
			DisableTimestamp:  true,
			DisableStacktrace: true,
			KeyGroupFields:    "group",
			FieldSorter:       sorter,
		}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger().With(
			logr.String("middle_name", "Thomas"),
			logr.String("last_name", "Wiggin"),
			logr.String("first_name", "Ender"),
		)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","group":{"first_name":"Ender","last_name":"Wiggin","middle_name":"Thomas"}}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("reverse sorted, three fields", func(t *testing.T) {
		formatterWithReverseSort := &formatters.JSON{DisableTimestamp: true, DisableStacktrace: true, FieldSorter: reverseSorter}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatterWithReverseSort, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger().With(
			logr.String("middle_name", "Thomas"),
			logr.String("last_name", "Wiggin"),
			logr.String("first_name", "Ender"),
		)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","middle_name":"Thomas","last_name":"Wiggin","first_name":"Ender"}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("reverse sorted, three fields, grouped", func(t *testing.T) {
		formatter := &formatters.JSON{
			DisableTimestamp:  true,
			DisableStacktrace: true,
			FieldSorter:       reverseSorter,
			KeyGroupFields:    "group",
		}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger().With(
			logr.String("middle_name", "Thomas"),
			logr.String("last_name", "Wiggin"),
			logr.String("first_name", "Ender"),
		)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		want := NL(`{"level":"error","msg":"This is an error.","group":{"middle_name":"Thomas","last_name":"Wiggin","first_name":"Ender"}}`)

		if strings.Compare(want, buf.String()) != 0 {
			t.Errorf("JSON does not match: expected %s   got %s", want, buf.String())
		}
	})

	t.Run("sorted, three fields, grouped, with caller", func(t *testing.T) {
		formatter := &formatters.JSON{
			DisableTimestamp:  true,
			DisableStacktrace: true,
			EnableCaller:      true,
			KeyGroupFields:    "group",
			FieldSorter:       sorter,
		}
		buf := &test.Buffer{}
		target := targets.NewWriterTarget(buf)
		err := lgr.AddTarget(target, "jsonTest", filter, formatter, 1000)
		require.NoError(t, err)

		logger := lgr.NewLogger().With(
			logr.String("middle_name", "Thomas"),
			logr.String("last_name", "Wiggin"),
			logr.String("first_name", "Ender"),
		)

		logger.Error("This is an error.")
		err = lgr.Flush()
		require.NoError(t, err)

		// {"level":"error","msg":"This is an error.","caller":"formatters/json_test.go:357","group":{"first_name":"Ender","last_name":"Wiggin","middle_name":"Thomas"}}
		want := regexp.MustCompile(`^{\"level\":\"error\",\"msg\":\"This is an error\.\",\"caller\":\"formatters/json_test.go:[0-9]+\",\"group\":{\"first_name\":\"Ender\",\"last_name\":\"Wiggin\",\"middle_name\":\"Thomas\"}}`)

		assert.Regexp(t, want, buf.String(), "JSON does not match")
	})

	err := lgr.Shutdown()
	require.NoError(t, err)
}

func sorter(fields []logr.Field) []logr.Field {
	cf := make([]logr.Field, len(fields))
	copy(cf, fields)

	sort.Sort(logr.FieldSorter(cf))
	return cf
}

func reverseSorter(fields []logr.Field) []logr.Field {
	cf := make([]logr.Field, len(fields))
	copy(cf, fields)

	sort.Sort(sort.Reverse(logr.FieldSorter(cf)))
	return cf
}

func NL(s string) string {
	return s + "\n"
}
