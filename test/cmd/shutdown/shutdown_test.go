package test

import (
	"testing"
	"time"

	"github.com/mattermost/logr"
)

func TestShutdown_NoTargetsAdded(t *testing.T) {
	lgr := &logr.Logr{MaxQueueSize: 1000}

	time.Sleep(2 * time.Second)

	err := lgr.Shutdown()
	if err != nil {
		t.Error(err)
	}
}
