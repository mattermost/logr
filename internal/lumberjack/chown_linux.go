//go:build linux

package lumberjack

import (
	"os"
	"syscall"
)

// osChown is a var so we can mock it out during tests.
var osChown = os.Chown

// chown sets the owner of the existing file name to that of info.
func chown(name string, info os.FileInfo) error {
	stat := info.Sys().(*syscall.Stat_t)
	return osChown(name, int(stat.Uid), int(stat.Gid))
}
