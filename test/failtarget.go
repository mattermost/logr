package test

import (
	"errors"

	"github.com/mattermost/logr"
)

// FailingTarget is a test target that always fails.
type FailingTarget struct {
	logr.Basic
}

// NewFailingTarget creates a target that always fails.
func NewFailingTarget(filter logr.Filter, formatter logr.Formatter) *FailingTarget {
	t := &FailingTarget{}
	t.Basic.Start(t, t, filter, formatter, 100)
	return t
}

// Write simply fails.
func (ft *FailingTarget) Write(rec *logr.LogRec) error {
	return errors.New("FailingTarget always fails")
}
