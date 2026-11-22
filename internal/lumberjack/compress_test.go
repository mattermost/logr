package lumberjack

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func trials(n int) int {
	if testing.Short() {
		return 2
	}
	return n
}

func makeLogContent(t *testing.T, size int) []byte {
	t.Helper()
	r := rand.New(rand.NewSource(1))
	var buf bytes.Buffer
	for buf.Len() < size {
		fmt.Fprintf(&buf, `{"level":"info","msg":"line %d","n":%d}`+"\n", buf.Len(), r.Int63())
	}
	return buf.Bytes()
}

func gunzipFile(t *testing.T, name string) []byte {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader %s: %v", name, err)
	}
	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("gunzip %s: %v", name, err)
	}
	return data
}

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestCompressConcurrentSameSource(t *testing.T) {
	content := makeLogContent(t, 1024*1024)

	for i := range trials(10) {
		dir := t.TempDir()
		src := filepath.Join(dir, "foo-2026-01-01T00-00-00.000.log")
		dst := src + compressSuffix
		if err := os.WriteFile(src, content, 0640); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		errs := make([]error, 2)
		for j := range errs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[j] = compressLogFile(src, dst)
			}()
		}
		wg.Wait()

		for j, err := range errs {
			// On Windows the first to finish cannot remove src while the other
			// still has it open; the other removes it, so only the result matters.
			if err != nil && runtime.GOOS != "windows" {
				t.Errorf("trial %d: compressor %d: %v", i, j, err)
			}
		}
		if names := dirNames(t, dir); len(names) != 1 || names[0] != filepath.Base(dst) {
			t.Fatalf("trial %d: want only %s, got %v", i, filepath.Base(dst), names)
		}
		if !bytes.Equal(gunzipFile(t, dst), content) {
			t.Fatalf("trial %d: decompressed content does not match source", i)
		}
	}
}

func TestCompressMissingSourceKeepsDst(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "foo-2026-01-01T00-00-00.000.log")
	dst := src + compressSuffix
	if err := os.WriteFile(dst, []byte("existing"), 0640); err != nil {
		t.Fatal(err)
	}

	if err := compressLogFile(src, dst); err != nil {
		t.Fatalf("want nil error for missing source, got %v", err)
	}
	existsWithContent(dst, []byte("existing"), t)
}

func TestRemoveStaleTemps(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{Filename: filepath.Join(dir, "foo.log"), Compress: true}

	stale := filepath.Join(dir, "foo-2026-01-01T00-00-00.000.log.gz.tmp111")
	fresh := filepath.Join(dir, "foo-2026-01-01T00-00-00.000.log.gz.tmp222")
	other := filepath.Join(dir, "bar-2026-01-01T00-00-00.000.log.gz.tmp333")
	for _, name := range []string{stale, fresh, other} {
		if err := os.WriteFile(name, []byte("partial"), 0640); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * staleTempAge)
	for _, name := range []string{stale, other} {
		if err := os.Chtimes(name, old, old); err != nil {
			t.Fatal(err)
		}
	}

	if err := l.millRunOnce(); err != nil {
		t.Fatal(err)
	}

	notExist(stale, t)
	exists(fresh, t)
	exists(other, t)
}

func TestRemoveStaleTempsKeepsBackupsWithTempLikeFilename(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{Filename: filepath.Join(dir, "foo.gz.tmp.log"), Compress: true}

	backup := filepath.Join(dir, "foo.gz.tmp-2026-01-01T00-00-00.000.log")
	backupGz := backup + compressSuffix
	stale := backupGz + tempInfix + "123"
	for _, name := range []string{backup, backupGz, stale} {
		if err := os.WriteFile(name, []byte("data"), 0640); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * staleTempAge)
	for _, name := range []string{backup, backupGz, stale} {
		if err := os.Chtimes(name, old, old); err != nil {
			t.Fatal(err)
		}
	}

	l.removeStaleTemps()

	exists(backup, t)
	exists(backupGz, t)
	notExist(stale, t)
}

func TestTimeFromNameOverlappingPrefixAndExt(t *testing.T) {
	l := &Logger{Filename: ".x.x-"}
	prefix, ext := l.prefixAndExt()
	equals(".x-", prefix, t)
	equals(".x-", ext, t)

	_, err := l.timeFromName(".x-", prefix, ext)
	notNil(err, t)
	equals(false, l.isTempName(".x-.gz.tmp123", prefix, ext), t)
}

func TestIsTempName(t *testing.T) {
	l := &Logger{Filename: "foo.log"}
	prefix, ext := l.prefixAndExt()
	for name, want := range map[string]bool{
		"foo-2026-01-01T00-00-00.000.log.gz.tmp123": true,
		"foo-2026-01-01T00-00-00.000.log.gz.tmp":    false,
		"foo-2026-01-01T00-00-00.000.log.gz.tmpabc": false,
		"foo-2026-01-01T00-00-00.000.log.gz":        false,
		"foo-2026-01-01T00-00-00.000.log":           false,
		"foo-notatime.log.gz.tmp123":                false,
		"bar-2026-01-01T00-00-00.000.log.gz.tmp123": false,
	} {
		if got := l.isTempName(name, prefix, ext); got != want {
			t.Errorf("isTempName(%q) = %v, want %v", name, got, want)
		}
	}
}

// TestReconfigureDuringCompress is the MM-72165 scenario: a second Logger for
// the same file starts milling while the first is still compressing a backup.
func TestReconfigureDuringCompress(t *testing.T) {
	currentTime = fakeTime
	megabyte = 1

	content := makeLogContent(t, 1024*1024)

	for i := range trials(5) {
		dir := t.TempDir()
		filename := logFile(dir)
		if err := os.WriteFile(filename, content, 0640); err != nil {
			t.Fatal(err)
		}
		newFakeTime()

		a := &Logger{Filename: filename, MaxSize: 10, Compress: true}
		if _, err := a.Write([]byte("a\n")); err != nil {
			t.Fatal(err)
		}
		a.Close()

		b := &Logger{Filename: filename, MaxSize: 10, Compress: true}
		if _, err := b.Write([]byte("b\n")); err != nil {
			t.Fatal(err)
		}
		closeAndWait(a)
		closeAndWait(b)

		var gzName string
		for _, name := range dirNames(t, dir) {
			switch {
			case name == filepath.Base(filename):
			case strings.HasSuffix(name, compressSuffix) && gzName == "":
				gzName = name
			default:
				t.Fatalf("trial %d: unexpected file %s in %v", i, name, dirNames(t, dir))
			}
		}
		if gzName == "" {
			t.Fatalf("trial %d: backup lost, dir has %v", i, dirNames(t, dir))
		}
		if !bytes.Equal(gunzipFile(t, filepath.Join(dir, gzName)), content) {
			t.Fatalf("trial %d: decompressed backup does not match original", i)
		}
	}
}
