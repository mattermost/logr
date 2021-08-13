package logr_test

import (
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetPackageName(t *testing.T) {
	pkgName := logr.GetPackageName("TestGetPackageName")

	assert.NotEmpty(t, pkgName, "pkgName should not be empty")
	t.Log("test pkg name is ", pkgName)
}

func TestGetLogrPackageName(t *testing.T) {
	pkgName := logr.GetLogrPackageName()

	assert.NotEmpty(t, pkgName, "pkgName should not be empty")
	t.Log("Logr pkg name is ", pkgName)
}
