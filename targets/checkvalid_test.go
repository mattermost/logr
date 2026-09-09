package targets

import (
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/require"
)

func TestFileOptionsCheckValid(t *testing.T) {
	t.Run("typical options are valid", func(t *testing.T) {
		fo := FileOptions{Filename: "/var/log/mattermost/mattermost.log", MaxSize: 100, MaxAge: 7, MaxBackups: 5}
		require.NoError(t, fo.CheckValid())
	})

	t.Run("filename is required", func(t *testing.T) {
		require.Error(t, FileOptions{}.CheckValid())
	})

	t.Run("filename is length limited", func(t *testing.T) {
		fo := FileOptions{Filename: strings.Repeat("a", logr.MaxFilePathLen+1)}
		require.Error(t, fo.CheckValid())
	})

	t.Run("filename rejects control characters", func(t *testing.T) {
		require.Error(t, FileOptions{Filename: "/var/log/mm\n.log"}.CheckValid())
	})

	t.Run("rotation options cannot be negative", func(t *testing.T) {
		require.Error(t, FileOptions{Filename: "a.log", MaxSize: -1}.CheckValid())
		require.Error(t, FileOptions{Filename: "a.log", MaxAge: -1}.CheckValid())
		require.Error(t, FileOptions{Filename: "a.log", MaxBackups: -1}.CheckValid())
	})
}

func TestTcpOptionsCheckValid(t *testing.T) {
	t.Run("typical options are valid", func(t *testing.T) {
		require.NoError(t, TcpOptions{Host: "logs.example.com", Port: 12201}.CheckValid())
	})

	t.Run("deprecated IP still works", func(t *testing.T) {
		require.NoError(t, TcpOptions{IP: "127.0.0.1", Port: 12201}.CheckValid())
	})

	t.Run("host and port are required", func(t *testing.T) {
		require.Error(t, TcpOptions{Port: 12201}.CheckValid())
		require.Error(t, TcpOptions{Host: "localhost"}.CheckValid())
	})

	t.Run("port is range checked", func(t *testing.T) {
		require.Error(t, TcpOptions{Host: "localhost", Port: -1}.CheckValid())
		require.Error(t, TcpOptions{Host: "localhost", Port: 65536}.CheckValid())
	})

	t.Run("host is length limited", func(t *testing.T) {
		to := TcpOptions{Host: strings.Repeat("h", logr.MaxHostnameLen+1), Port: 12201}
		require.Error(t, to.CheckValid())
	})

	t.Run("cert is length limited", func(t *testing.T) {
		to := TcpOptions{Host: "localhost", Port: 12201, Cert: strings.Repeat("c", logr.MaxCertLen+1)}
		require.Error(t, to.CheckValid())
	})
}
