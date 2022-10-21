// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package targets

import (
	"testing"

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
}
