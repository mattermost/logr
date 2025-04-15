// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package targets

import (
	"testing"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/test"
	"github.com/stretchr/testify/require"
)

const (
	Server   = "localhost"
	TestPort = 18067
)

func TestNewTcpTarget(t *testing.T) {
	opt := logr.OnLoggerError(func(err error) {
		t.Error("OnLoggerError", err)
	})
	lgr, _ := logr.New(opt)

	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	formatter := &formatters.JSON{}
	opts := &TcpOptions{
		IP:   Server,
		Port: TestPort,
	}

	logger := lgr.NewLogger().With(logr.String("name", "wiggin"))

	t.Run("TCP logging", func(t *testing.T) {
		buf := &test.Buffer{}
		server, err := test.NewSocketServer(TestPort, buf)
		require.NoError(t, err)

		tcp := NewTcpTarget(opts)

		err = lgr.AddTarget(tcp, "tcp_test", filter, formatter, 1000)
		require.NoError(t, err)

		data := []string{"I drink your milkshake!", "We don't need no badges!", "You can't fight in here. This is the war room!"}

		for _, s := range data {
			logger.Info(s)
		}

		err = logger.Logr().Shutdown()
		require.NoError(t, err)

		err = server.WaitForAnyConnection()
		require.NoError(t, err)

		err = server.StopServer(true)
		require.NoError(t, err)

		sdata := buf.String()
		for _, s := range data {
			require.Contains(t, sdata, s)
		}
	})

	t.Run("TCP connection recovery", func(t *testing.T) {
		// Use a different port for the recovery test to avoid conflicts
		recoveryPort := TestPort + 1
		recoveryOpts := &TcpOptions{
			IP:   Server,
			Port: recoveryPort,
		}

		// First start with no server running
		recoveryLgr, _ := logr.New()
		tcp := NewTcpTarget(recoveryOpts)

		err := recoveryLgr.AddTarget(tcp, "tcp_recovery_test", filter, formatter, 1000)
		require.NoError(t, err)

		recoveryLogger := recoveryLgr.NewLogger().With(logr.String("name", "recovery"))

		// Try to log something when no server is running (will be queued)
		recoveryLogger.Info("buffered")

		// Now start the server
		buf := &test.Buffer{}
		server, err := test.NewSocketServer(recoveryPort, buf)
		require.NoError(t, err)

		// Flush logs to ensure delivery
		err = recoveryLgr.Flush()
		require.NoError(t, err)

		// Wait for connection to be established
		err = server.WaitForAnyConnection()
		require.NoError(t, err)

		// Log a message after starting
		recoveryLogger.Info("pre-stop")

		// Flush to ensure messages are processed
		err = recoveryLgr.Flush()
		require.NoError(t, err)

		// Short wait for flush to settle
		time.Sleep(100 * time.Millisecond)

		// Verify messages were received
		sdata := buf.String()
		require.Contains(t, sdata, "buffered")
		require.Contains(t, sdata, "pre-stop")

		// Close the server to simulate connection loss
		err = server.StopServer(false)
		require.NoError(t, err)

		// Wait for all connections to close
		time.Sleep(1 * time.Second)

		// Try logging with server down
		recoveryLogger.Info("during-stop")

		// Wait to ensure we try to log while server is down
		// Important: this delay needs to be long enough to
		// trigger at least one call to tcp dial()
		time.Sleep(500 * time.Millisecond)

		// Start server again to test reconnection
		buf2 := &test.Buffer{}
		server2, err := test.NewSocketServer(recoveryPort, buf2)
		require.NoError(t, err)

		// Wait for any connections
		err = server2.WaitForAnyConnection()
		require.NoError(t, err)

		// Try logging with server up again
		recoveryLogger.Info("post-stop")

		// Flush logs to ensure delivery
		err = recoveryLgr.Flush()
		require.NoError(t, err)

		// Short wait for flush to settle
		time.Sleep(100 * time.Millisecond)

		// Verify at least some messages got through
		sdata2 := buf2.String()
		require.Contains(t, sdata2, "during-stop")
		require.Contains(t, sdata2, "post-stop")

		// Clean up
		err = recoveryLgr.Shutdown()
		require.NoError(t, err)

		err = server2.StopServer(false)
		require.NoError(t, err)
	})
}
