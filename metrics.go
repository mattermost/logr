package logr

import (
	"fmt"
	"sync/atomic"
)

// SizeAndCap represents the metrics of something that has a current size
// and capacity such as a channel or buffer.
type SizeAndCap struct {
	Size int
	Cap  int
}

// TargetMetrics provides a snapshot of a target's metrics such as
// current queue size and capacity.
type TargetMetrics struct {
	Queue       SizeAndCap
	LoggedCount uint64
}

// TargetWithMetrics is a target that provides metrics.
type TargetWithMetrics interface {
	GetMetrics() TargetMetrics
}

// Metrics provides a snapshot of Logr metrics, including queue sizes.
type Metrics struct {
	MainQueue   SizeAndCap
	LoggedCount uint64
	ErrorCount  uint64

	Targets map[string]TargetMetrics
}

// GetMetrics returns a snapshot of current logging metrics.
func (logr *Logr) GetMetrics() Metrics {
	metrics := Metrics{}

	metrics.LoggedCount = atomic.LoadUint64(&logr.loggedCount)
	metrics.ErrorCount = atomic.LoadUint64(&logr.errorCount)
	metrics.MainQueue = SizeAndCap{
		Size: len(logr.in),
		Cap:  cap(logr.in),
	}
	metrics.Targets = make(map[string]TargetMetrics)

	logr.tmux.RLock()
	defer logr.tmux.RUnlock()
	for _, target := range logr.targets {
		if tt, ok := target.(TargetWithMetrics); ok {
			name := fmt.Sprintf("%v", target)
			metrics.Targets[name] = tt.GetMetrics()
		}
	}
	return metrics
}
