package config

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/logr/v2"
)

// TestReconfigureDuringCompressKeepsBackup covers MM-72165: reconfiguring while
// a rotated backup is being compressed must not delete the backup and its .gz.
func TestReconfigureDuringCompressKeepsBackup(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "app.log")

	r := rand.New(rand.NewSource(1))
	var buf bytes.Buffer
	for buf.Len() < 4*1024*1024 {
		fmt.Fprintf(&buf, `{"level":"info","msg":"before restart","n":%d}`+"\n", r.Int63())
	}
	original := buf.Bytes()
	require.NoError(t, os.WriteFile(filename, original, 0600))

	opts, err := json.Marshal(map[string]any{"filename": filename, "max_size": 1, "compress": true})
	require.NoError(t, err)
	cfg := map[string]TargetCfg{
		"file": {
			Type:    "file",
			Options: opts,
			Format:  "plain",
			Levels:  []logr.Level{logr.Info},
		},
	}

	lgr, err := logr.New()
	require.NoError(t, err)
	logger := lgr.NewLogger()

	require.NoError(t, ConfigureTargets(lgr, cfg, nil))
	logger.Info("first configure")
	require.NoError(t, lgr.Flush())
	require.NoError(t, ConfigureTargets(lgr, cfg, nil))
	logger.Info("second configure")
	require.NoError(t, lgr.Shutdown())

	var names []string
	require.Eventually(t, func() bool {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return false
		}
		names = names[:0]
		for _, e := range entries {
			names = append(names, e.Name())
		}
		for _, name := range names {
			if name != "app.log" && !strings.HasSuffix(name, ".gz") {
				return false
			}
		}
		return true
	}, 10*time.Second, 20*time.Millisecond, "compression did not finish: %v", names)

	var gzName string
	for _, name := range names {
		if strings.HasSuffix(name, ".gz") {
			require.Empty(t, gzName, "more than one backup: %v", names)
			gzName = name
		}
	}
	require.NotEmpty(t, gzName, "backup lost: %v", names)

	f, err := os.Open(filepath.Join(dir, gzName))
	require.NoError(t, err)
	defer f.Close()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	got, err := io.ReadAll(gz)
	require.NoError(t, err)
	require.True(t, bytes.Equal(original, got), "decompressed backup does not match original")
}
