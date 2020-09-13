package test

import (
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/require"
)

func TestShutdown_NoTargetsAdded(t *testing.T) {
	opt := logr.MaxQueueSize(1000)

	lgr, err := logr.New(opt)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	err = lgr.Shutdown()
	require.NoError(t, err)
}
