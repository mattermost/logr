//go:build !windows && !nacl && !plan9

package targets

import (
	"strings"
	"testing"

	"github.com/mattermost/logr/v2"
	"github.com/stretchr/testify/require"
)

func TestSyslogOptionsCheckValid(t *testing.T) {
	t.Run("remote options are valid", func(t *testing.T) {
		require.NoError(t, SyslogOptions{Host: "logs.example.com", Port: 514, Tag: "mattermost"}.CheckValid())
	})

	t.Run("deprecated IP still works", func(t *testing.T) {
		require.NoError(t, SyslogOptions{IP: "127.0.0.1", Port: 514}.CheckValid())
	})

	t.Run("local syslog needs neither host nor port", func(t *testing.T) {
		// Init connects to local syslog with an empty address when both are
		// unset, and the schema documents port 0 as meaning local.
		require.NoError(t, SyslogOptions{}.CheckValid())
		require.NoError(t, SyslogOptions{Tag: "logrtest"}.CheckValid())
	})

	t.Run("a host without a port is rejected", func(t *testing.T) {
		err := SyslogOptions{Host: "logs.example.com"}.CheckValid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing port")
	})

	t.Run("a port without a host is rejected", func(t *testing.T) {
		err := SyslogOptions{Port: 514}.CheckValid()
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing host")
	})

	t.Run("port is range checked", func(t *testing.T) {
		require.Error(t, SyslogOptions{Host: "localhost", Port: -1}.CheckValid())
		require.Error(t, SyslogOptions{Host: "localhost", Port: 65536}.CheckValid())
	})

	t.Run("host is length limited", func(t *testing.T) {
		so := SyslogOptions{Host: strings.Repeat("h", logr.MaxHostnameLen+1), Port: 514}
		require.Error(t, so.CheckValid())
	})

	t.Run("host rejects control characters", func(t *testing.T) {
		require.Error(t, SyslogOptions{Host: "logs\n.example.com", Port: 514}.CheckValid())
	})

	t.Run("tag is length limited", func(t *testing.T) {
		so := SyslogOptions{Host: "localhost", Port: 514, Tag: strings.Repeat("t", logr.MaxTagLen+1)}
		require.Error(t, so.CheckValid())
	})

	t.Run("tag rejects control characters", func(t *testing.T) {
		require.Error(t, SyslogOptions{Host: "localhost", Port: 514, Tag: "mm\nforged"}.CheckValid())
	})

	t.Run("cert is length limited", func(t *testing.T) {
		so := SyslogOptions{Host: "localhost", Port: 514, Cert: strings.Repeat("c", logr.MaxCertLen+1)}
		require.Error(t, so.CheckValid())
	})
}

func TestSyslogOptionsGetHost(t *testing.T) {
	require.Equal(t, "logs.example.com", SyslogOptions{Host: "logs.example.com", IP: "10.0.0.1"}.GetHost())
	require.Equal(t, "10.0.0.1", SyslogOptions{IP: "10.0.0.1"}.GetHost())
	require.Empty(t, SyslogOptions{}.GetHost(), "no host means local syslog")
}
