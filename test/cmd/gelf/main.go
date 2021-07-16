package main

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/mattermost/logr/v2"
	"github.com/mattermost/logr/v2/formatters"
	"github.com/mattermost/logr/v2/targets"
)

// Settings
const (
	Server = "192.168.1.68"
	Port   = 12201

	Loops = 100
	QSIZE = 1000
)

var (
	errorCount           uint32
	queueFullCount       uint32
	targetQueueFullCount uint32
)

func handleLoggerError(err error) {
	atomic.AddUint32(&errorCount, 1)
	fmt.Fprintln(os.Stderr, "!!!!! OnLoggerError -- ", err)
}

func handleQueueFull(rec *logr.LogRec, maxQueueSize int) bool {
	fmt.Fprintf(os.Stderr, "!!!!! OnQueueFull - Max size %d. Count %d. Blocking...\n",
		maxQueueSize, atomic.AddUint32(&queueFullCount, 1))
	return false
}

func handleTargetQueueFull(target logr.Target, rec *logr.LogRec, maxQueueSize int) bool {
	fmt.Fprintf(os.Stderr, "!!!!! OnTargetQueueFull - (%v). Max size %d. Count %d. Blocking...\n",
		target, maxQueueSize, atomic.AddUint32(&targetQueueFullCount, 1))
	return false
}

func main() {
	lgr, err := logr.New(
		logr.MaxQueueSize(QSIZE),
		logr.OnLoggerError(handleLoggerError),
		logr.OnQueueFull(handleQueueFull),
		logr.OnTargetQueueFull(handleTargetQueueFull),
	)
	if err != nil {
		panic(err)
	}

	// create TCP target to server supporting GELF
	filter := &logr.StdFilter{Lvl: logr.Info, Stacktrace: logr.Error}
	formatter := &formatters.Gelf{EnableCaller: true}

	params := &targets.TcpOptions{
		IP:   Server,
		Port: Port,
	}

	tcp := targets.NewTcpTarget(params)

	err = lgr.AddTarget(tcp, "tcp_test", filter, formatter, QSIZE)
	if err != nil {
		panic(err)
	}

	logger := lgr.NewLogger().With(logr.String("name", "Wiggin"))

	for i := 0; i < Loops; i++ {
		if i%10 == 0 {
			logger.Error("This is an error!")
			continue
		}
		logger.Info("This is a message")
	}

	time.Sleep(time.Second * 3)

	err = lgr.Shutdown()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Fprintf(os.Stdout, "Exiting normally. loops=%d, errors=%d, queueFull=%d, targetFull=%d\n",
		Loops,
		atomic.LoadUint32(&errorCount),
		atomic.LoadUint32(&queueFullCount),
		atomic.LoadUint32(&targetQueueFullCount),
	)
}
