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

	t.Run("metrics after AddTarget should fail", func(t *testing.T) {
		lgr := &logr.Logr{}
		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// create target
		buf := &bytes.Buffer{}
		tgt := target.NewWriterTarget(filter, formatter, buf, 100)
		tgt.SetName(TestTargetName)

		// add target before adding metrics
		err := lgr.AddTarget(tgt)
		require.NoError(t, err)

		collector := test.NewTestMetricsCollector()
		err = lgr.SetMetricsCollector(collector)
		require.Error(t, err)
	})

	t.Run("metrics before AddTarget should pass", func(t *testing.T) {
		lgr := &logr.Logr{}
		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// Add metrics before AddTarget
		collector := test.NewTestMetricsCollector()
		err := lgr.SetMetricsCollector(collector)
		require.NoError(t, err)

		// Create target
		buf := &bytes.Buffer{}
		tgt := target.NewWriterTarget(filter, formatter, buf, 100)
		tgt.SetName(TestTargetName)

		err = lgr.AddTarget(tgt)
		require.NoError(t, err)

		logger := lgr.NewLogger()
		logger.Info("Say 'hello' to my little friend!")
		logger.Info("Hasta la vista, baby.")

		err = lgr.Flush()
		require.NoError(t, err)

		metricsLogr := collector.Get("_logr")
		metricsTarget := collector.Get(TestTargetName)

		require.EqualValues(t, 2, metricsLogr.Logged)
		require.EqualValues(t, 2, metricsTarget.Logged)

		require.EqualValues(t, 0, metricsLogr.Errors)
		require.EqualValues(t, 0, metricsTarget.Errors)
	})

	t.Run("metrics with failing target", func(t *testing.T) {
		lgr := &logr.Logr{}
		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// Add metrics before AddTarget
		collector := test.NewTestMetricsCollector()
		err := lgr.SetMetricsCollector(collector)
		require.NoError(t, err)

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
		lgr := &logr.Logr{}
		defer func() {
			err := lgr.Shutdown()
			require.NoError(t, err)
		}()

		// Add metrics before AddTarget
		collector := test.NewTestMetricsCollector()
		err := lgr.SetMetricsCollector(collector)
		require.NoError(t, err)

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
