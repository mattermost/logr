package logr_test

import (
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
	"github.com/stretchr/testify/require"
)

func TestLogr_RedirectStdLog(t *testing.T) {
	buf := &bytes.Buffer{}

	lgr, err := logr.New(logr.StackFilter("log"))
	require.NoError(t, err)

	filter := &logr.StdFilter{logr.Info, logr.Error}
	formatter := &formatters.Plain{Delim: " | "}

	tgt := targets.NewWriterTarget(buf)
	err = lgr.AddTarget(tgt, "buf", filter, formatter, 1000)
	require.NoError(t, err)

	tgt = targets.NewWriterTarget(os.Stdout)
	err = lgr.AddTarget(tgt, "stdout", filter, formatter, 1000)
	require.NoError(t, err)

	// remember old settings.
	flags := log.Flags()
	prefix := log.Prefix()

	restoreFunc := lgr.RedirectStdLog()

	log.Println("Peaky Blinders!")

	err = lgr.Flush()
	require.NoError(t, err)

	require.Contains(t, buf.String(), "Peaky Blinders!")

	err = lgr.Shutdown()
	require.NoError(t, err)

	restoreFunc()

	// check settings restored
	require.Equal(t, flags, log.Flags())
	require.Equal(t, prefix, log.Prefix())
}
