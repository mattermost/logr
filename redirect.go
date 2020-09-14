package logr

import (
	"log"
	"os"
	"strings"
)

// RedirectStdLog redirects output from the standard library's package-global logger
// to this logger at level Info. Since Logr already handles caller annotations, timestamps, etc.,
// it automatically disables the standard library's annotations and prefixing.
// A function is returned that restores the original prefix and flags and resets the standard
// library's output to os.Stderr.
func (logr *Logr) RedirectStdLog() func() {
	flags := log.Flags()
	prefix := log.Prefix()
	log.SetFlags(0)
	log.SetPrefix("")

	logger := logr.NewLogger().WithField("src", "stdlog")
	adapter := newStdLogAdapter(logger)
	log.SetOutput(adapter)

	return func() {
		log.SetFlags(flags)
		log.SetPrefix(prefix)
		log.SetOutput(os.Stderr)
	}
}

type stdLogAdapter struct {
	logger Logger
}

func newStdLogAdapter(logger Logger) *stdLogAdapter {
	return &stdLogAdapter{
		logger: logger,
	}
}

// Write implements io.Writer
func (a *stdLogAdapter) Write(p []byte) (int, error) {
	s := strings.TrimSpace(string(p))
	a.logger.Info(s)
	return len(p), nil
}
