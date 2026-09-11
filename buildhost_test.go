package logr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type queueProbeTarget struct{}

func (t *queueProbeTarget) Init() error                              { return nil }
func (t *queueProbeTarget) Write(p []byte, rec *LogRec) (int, error) { return len(p), nil }
func (t *queueProbeTarget) Shutdown() error                          { return nil }

// TestBuildHostQueueSize pins the queue capacity buildHost produces, which is
// the only place the non-positive-means-default rule is observable.
func TestBuildHostQueueSize(t *testing.T) {
	cases := []struct {
		name string
		size int
		want int
	}{
		{"explicit size is used", 25, 25},
		{"unset uses the default", 0, DefaultMaxQueueSize},
		{"negative uses the default", -1, DefaultMaxQueueSize},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lgr, err := New()
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = lgr.Shutdown()
			})

			host, err := lgr.buildHost(TargetSpec{
				Target:       &queueProbeTarget{},
				Name:         tc.name,
				Filter:       &StdFilter{Lvl: Info},
				MaxQueueSize: tc.size,
			})
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = host.Shutdown(context.Background())
			})

			require.Equal(t, tc.want, cap(host.in))
		})
	}
}
