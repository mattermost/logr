//go:build !windows && !nacl && !plan9

package targets

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSyslogOptionsGetHost(t *testing.T) {
	require.Equal(t, "logs.example.com", SyslogOptions{Host: "logs.example.com", IP: "10.0.0.1"}.GetHost())
	require.Equal(t, "10.0.0.1", SyslogOptions{IP: "10.0.0.1"}.GetHost())
	require.Empty(t, SyslogOptions{}.GetHost(), "no host means local syslog")
}

func TestSyslogOptionsCheckValid(t *testing.T) {
	t.Run("no host and no port means local syslog", func(t *testing.T) {
		require.NoError(t, SyslogOptions{}.CheckValid())
	})

	t.Run("remote options are valid", func(t *testing.T) {
		require.NoError(t, SyslogOptions{Host: "logs.example.com", Port: 514}.CheckValid())
	})

	t.Run("host without a port is rejected", func(t *testing.T) {
		require.Error(t, SyslogOptions{Host: "logs.example.com"}.CheckValid())
	})

	t.Run("port out of range is rejected", func(t *testing.T) {
		require.Error(t, SyslogOptions{Host: "logs.example.com", Port: -1}.CheckValid())
		require.Error(t, SyslogOptions{Host: "logs.example.com", Port: 65536}.CheckValid())
	})
}
