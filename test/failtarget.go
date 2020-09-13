package test

import (
	"errors"

	"github.com/mattermost/logr/v2"
)

// FailingTarget is a test target that always fails.
type FailingTarget struct {
}

// NewFailingTarget creates a target that always fails.
func NewFailingTarget() *FailingTarget {
	t := &FailingTarget{}
	return t
}

func (ft *FailingTarget) Init() error {
	return nil
}

// Write simply fails.
func (ft *FailingTarget) Write(p []byte, rec *logr.LogRec) (int, error) {
	return 0, errors.New("FailingTarget always fails")
}

func (ft *FailingTarget) Shutdown() error {
	return nil
}
