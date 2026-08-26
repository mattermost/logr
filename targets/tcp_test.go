// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package targets

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
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

		// Flush to ensure logs are sent
		err = logger.Logr().Flush()
		require.NoError(t, err)

		// Give time for TCP connection to be established (TCP connects asynchronously)
		time.Sleep(500 * time.Millisecond)

		// Wait for connection to be established
		err = server.WaitForAnyConnection()
		require.NoError(t, err)

		err = logger.Logr().Shutdown()
		require.NoError(t, err)

		err = server.StopServer(false)
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

	t.Run("TCP connection errors are throttled", func(t *testing.T) {
		var connErrCount int32

		opt := logr.OnLoggerError(func(err error) {
			if strings.Contains(err.Error(), "connection error") {
				atomic.AddInt32(&connErrCount, 1)
			}
		})
		throttleLgr, err := logr.New(opt)
		require.NoError(t, err)

		// Grab a port and immediately release it so nothing is listening on it;
		// every dial attempt against it will fail immediately.
		freeListener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		throttlePort := freeListener.Addr().(*net.TCPAddr).Port
		require.NoError(t, freeListener.Close())

		throttleOpts := &TcpOptions{
			IP:   Server,
			Port: throttlePort,
		}
		tcp := NewTcpTarget(throttleOpts)

		// Registered here, not at the end, so the retry goroutine is stopped
		// even if an assertion below fails and returns early. Tcp.Shutdown is
		// idempotent, so this is safe regardless of whether ShutdownWithTimeout
		// already reached it.
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			_ = throttleLgr.ShutdownWithTimeout(ctx)
			_ = tcp.Shutdown()
		})

		err = throttleLgr.AddTarget(tcp, "tcp_throttle_test", filter, formatter, 1000)
		require.NoError(t, err)

		throttleLogger := throttleLgr.NewLogger().With(logr.String("name", "throttle"))
		throttleLogger.Info("this will retry against a closed port")

		// Attempt 1 is reported immediately.
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&connErrCount) == 1
		}, time.Second, 10*time.Millisecond, "expected the first connection error to be reported")

		// Attempts 2-9 are throttled; backoff (100ms, growing 1.5x per retry)
		// reaches attempt 10 after ~7.5s cumulative sleep.
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&connErrCount) == 2
		}, 10*time.Second, 50*time.Millisecond, "expected attempt 10 to be reported")

		// Attempt 11 follows after another ~3.8s backoff; confirm it's throttled
		// rather than just checking too early to have seen it fail.
		time.Sleep(5 * time.Second)
		got := atomic.LoadInt32(&connErrCount)
		require.EqualValues(t, 2, got, "expected attempt 11 to be throttled, got %d reports", got)
	})

	t.Run("Shutdown is idempotent", func(t *testing.T) {
		tcp := NewTcpTarget(&TcpOptions{IP: Server, Port: TestPort + 3})
		require.NoError(t, tcp.Shutdown())
		require.NotPanics(t, func() {
			require.NoError(t, tcp.Shutdown())
		})
	})
}
