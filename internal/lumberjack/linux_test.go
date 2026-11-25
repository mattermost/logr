//go:build linux

package lumberjack

import (
	"os"
	"sync"
	"syscall"
	"testing"
)

func TestMaintainMode(t *testing.T) {
	currentTime = fakeTime
	dir := makeTempDir("TestMaintainMode", t)
	defer os.RemoveAll(dir)

	filename := logFile(dir)

	mode := os.FileMode(0600)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, mode)
	isNil(err, t)
	f.Close()

	l := &Logger{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer closeAndWait(l)
	b := []byte("boo!")
	n, err := l.Write(b)
	isNil(err, t)
	equals(len(b), n, t)

	newFakeTime()

	err = l.Rotate()
	isNil(err, t)

	filename2 := backupFile(dir)
	info, err := os.Stat(filename)
	isNil(err, t)
	info2, err := os.Stat(filename2)
	isNil(err, t)
	equals(mode, info.Mode(), t)
	equals(mode, info2.Mode(), t)
}

func TestMaintainOwner(t *testing.T) {
	fakeFS := newFakeFS()
	osChown = fakeFS.Chown
	osStat = fakeFS.Stat
	defer func() {
		osChown = os.Chown
		osStat = os.Stat
	}()
	currentTime = fakeTime
	dir := makeTempDir("TestMaintainOwner", t)
	defer os.RemoveAll(dir)

	filename := logFile(dir)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	isNil(err, t)
	f.Close()

	l := &Logger{
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer closeAndWait(l)
	b := []byte("boo!")
	n, err := l.Write(b)
	isNil(err, t)
	equals(len(b), n, t)

	newFakeTime()

	err = l.Rotate()
	isNil(err, t)

	equals(fakeFile{uid: 555, gid: 666}, fakeFS.owner(filename, t), t)
}

func TestCompressMaintainMode(t *testing.T) {
	currentTime = fakeTime

	dir := makeTempDir("TestCompressMaintainMode", t)
	defer os.RemoveAll(dir)

	filename := logFile(dir)

	mode := os.FileMode(0600)
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, mode)
	isNil(err, t)
	f.Close()

	l := &Logger{
		Compress:   true,
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer closeAndWait(l)
	b := []byte("boo!")
	n, err := l.Write(b)
	isNil(err, t)
	equals(len(b), n, t)

	newFakeTime()

	err = l.Rotate()
	isNil(err, t)

	// we need to wait a little bit since the files get compressed on a different
	// goroutine.
	waitMill(l)

	// a compressed version of the log file should now exist with the correct
	// mode.
	filename2 := backupFile(dir)
	info, err := os.Stat(filename)
	isNil(err, t)
	info2, err := os.Stat(filename2 + compressSuffix)
	isNil(err, t)
	equals(mode, info.Mode(), t)
	equals(mode, info2.Mode(), t)
}

func TestCompressMaintainOwner(t *testing.T) {
	fakeFS := newFakeFS()
	osChown = fakeFS.Chown
	osStat = fakeFS.Stat
	defer func() {
		osChown = os.Chown
		osStat = os.Stat
	}()
	currentTime = fakeTime
	dir := makeTempDir("TestCompressMaintainOwner", t)
	defer os.RemoveAll(dir)

	filename := logFile(dir)

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	isNil(err, t)
	f.Close()

	l := &Logger{
		Compress:   true,
		Filename:   filename,
		MaxBackups: 1,
		MaxSize:    100, // megabytes
	}
	defer closeAndWait(l)
	b := []byte("boo!")
	n, err := l.Write(b)
	isNil(err, t)
	equals(len(b), n, t)

	newFakeTime()

	err = l.Rotate()
	isNil(err, t)

	waitMill(l)

	// a compressed version of the log file should now exist with the correct
	// owner.
	filename2 := backupFile(dir)
	equals(fakeFile{uid: 555, gid: 666}, fakeFS.owner(filename2+compressSuffix, t), t)
}

type fakeFile struct {
	uid int
	gid int
}

// fakeFS records chowns by inode so they follow a file across renames.
type fakeFS struct {
	mu    sync.Mutex
	files map[uint64]fakeFile
}

func newFakeFS() *fakeFS {
	return &fakeFS{files: make(map[uint64]fakeFile)}
}

func (fs *fakeFS) Chown(name string, uid, gid int) error {
	info, err := os.Stat(name)
	if err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.files[info.Sys().(*syscall.Stat_t).Ino] = fakeFile{uid: uid, gid: gid}
	return nil
}

func (fs *fakeFS) owner(name string, t testing.TB) fakeFile {
	info, err := os.Stat(name)
	isNilUp(err, t, 1)
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.files[info.Sys().(*syscall.Stat_t).Ino]
}

func (fs *fakeFS) Stat(name string) (os.FileInfo, error) {
	info, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	stat := info.Sys().(*syscall.Stat_t)
	stat.Uid = 555
	stat.Gid = 666
	return info, nil
}
