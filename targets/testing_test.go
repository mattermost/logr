package targets

import (
	"testing"

	"github.com/mattermost/logr/v2"
)

func TestCreateTestLogger(t *testing.T) {
	logger, shutdown := CreateTestLogger(t, logr.Debug, logr.Info)
	defer shutdown()

	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			logger.Debug("counting even", logr.Int("count", i))
		} else {
			logger.Info("counting odd", logr.Int("count", i))
		}
	}
}
