package logr_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/logr"
	"github.com/mattermost/logr/format"
	"github.com/mattermost/logr/target"
	"github.com/mattermost/logr/test"
	"github.com/stretchr/testify/require"
)

const (
	TestTargetName = "test_target"
)

func TestLogr_SetMetricsCollector(t *testing.T) {
	formatter := &format.Plain{DisableTimestamp: true, Delim: " | "}
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}

	t.Run("adding metrics should pass", func(t *testing.T) {
		collector := test.NewTestMetricsCollector()
		opt := logr.SetMetricsCollector(collector, 1000)

		lgr, err := logr.New(opt)
		require.NoError(t, err)

		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// Create target
		buf := &bytes.Buffer{}
		tgt := target.NewWriterTarget(filter, formatter, buf, 100)
		tgt.SetName(TestTargetName)

		err = lgr.AddTarget(tgt)
		require.NoError(t, err)

		logger := lgr.NewLogger()
		logger.Info("These go to eleven.")
		logger.Info("Pay no attention to that man behind the curtain!")

		err = lgr.Flush()
		require.NoError(t, err)

		metricsLogr := collector.Get("_logr")
		metricsTarget := collector.Get(TestTargetName)

		require.EqualValues(t, 2, metricsLogr.Logged)
		require.EqualValues(t, 2, metricsTarget.Logged)

		require.EqualValues(t, 0, metricsLogr.Errors)
		require.EqualValues(t, 0, metricsTarget.Errors)
	})

	t.Run("adding nil metrics should fail", func(t *testing.T) {
		opt := logr.SetMetricsCollector(nil, 1000)

		lgr, err := logr.New(opt)
		require.Error(t, err)
		require.Nil(t, lgr)
	})

	t.Run("metrics with failing target", func(t *testing.T) {
		collector := test.NewTestMetricsCollector()
		opt := logr.SetMetricsCollector(collector, 1000)

		lgr, err := logr.New(opt)
		require.NoError(t, err)

		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// Create target
		tgt := test.NewFailingTarget(filter, formatter)
		tgt.SetName(TestTargetName)

		err = lgr.AddTarget(tgt)
		require.NoError(t, err)

		logger := lgr.NewLogger()
		logger.Info("You're gonna need a bigger boat.")
		logger.Info("I see dead people.")

		err = lgr.Flush()
		require.NoError(t, err)

		metricsLogr := collector.Get("_logr")
		metricsTarget := collector.Get(TestTargetName)

		require.EqualValues(t, 2, metricsLogr.Logged)
		require.EqualValues(t, 0, metricsTarget.Logged)

		require.EqualValues(t, 2, metricsLogr.Errors)
		require.EqualValues(t, 2, metricsTarget.Errors)
	})

	t.Run("metrics with multiple targets", func(t *testing.T) {
		collector := test.NewTestMetricsCollector()
		opt := logr.SetMetricsCollector(collector, 1000)

		lgr, err := logr.New(opt)
		require.NoError(t, err)
		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// Create targets
		buf1 := &bytes.Buffer{}
		buf2 := &bytes.Buffer{}
		tgt1 := target.NewWriterTarget(filter, formatter, buf1, 100)
		tgt2 := target.NewWriterTarget(filter, formatter, buf2, 100)
		tgt1.SetName(TestTargetName + "1")
		tgt2.SetName(TestTargetName + "2")

		err = lgr.AddTarget(tgt1)
		require.NoError(t, err)
		err = lgr.AddTarget(tgt2)
		require.NoError(t, err)

		logger := lgr.NewLogger()
		logger.Info("What we've got here is a failure to communicate.")
		logger.Info("I love the smell of napalm in the morning.")

		err = lgr.Flush()
		require.NoError(t, err)

		metricsLogr := collector.Get("_logr")
		metricsTarget1 := collector.Get(TestTargetName + "1")
		metricsTarget2 := collector.Get(TestTargetName + "2")

		require.EqualValues(t, 2, metricsLogr.Logged)
		require.EqualValues(t, 2, metricsTarget1.Logged)
		require.EqualValues(t, 2, metricsTarget2.Logged)

		require.EqualValues(t, 0, metricsLogr.Errors)
		require.EqualValues(t, 0, metricsTarget1.Errors)
		require.EqualValues(t, 0, metricsTarget2.Errors)
	})
}
