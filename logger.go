package logr

import (
	"fmt"
	"sync"
)

var loggerPool = sync.Pool{}

// Logger implements Logr APIs.
// TODO expand docs for this key struct
type Logger struct {
	fields Fields
	parent *Logger
}

// NewLogger creates a logger using defaults. A logger is light-weight
// enough to create on-demand, but typically one or more loggers are
// created and re-used.
func NewLogger() *Logger {
	logger := &Logger{}
	return logger
}

// WithField creates a new Logger with any existing fields
// plus the new one.
func (logger *Logger) WithField(key string, value interface{}) *Logger {
	return logger.WithFields(Fields{key: value})
}

// WithFields creates a new Logger with any existing fields
// plus the new ones.
func (logger *Logger) WithFields(fields Fields) *Logger {
	l := &Logger{fields: Fields{}, parent: logger}
	for k, v := range logger.fields {
		l.fields[k] = v
	}
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

func (logger *Logger) Log(level Level, args ...interface{}) {
	if entry.Logger.IsLevelEnabled(level) {
		entry.log(level, fmt.Sprint(args...))
	}
}

func (logger *Logger) Trace(args ...interface{}) {
	entry.Log(TraceLevel, args...)
}

func (logger *Logger) Debug(args ...interface{}) {
	entry.Log(DebugLevel, args...)
}

func (logger *Logger) Print(args ...interface{}) {
	entry.Info(args...)
}

func (logger *Logger) Info(args ...interface{}) {
	entry.Log(InfoLevel, args...)
}

func (logger *Logger) Warn(args ...interface{}) {
	entry.Log(WarnLevel, args...)
}

func (logger *Logger) Warning(args ...interface{}) {
	entry.Warn(args...)
}

func (logger *Logger) Error(args ...interface{}) {
	entry.Log(ErrorLevel, args...)
}

func (logger *Logger) Fatal(args ...interface{}) {
	entry.Log(FatalLevel, args...)
	entry.Logger.Exit(1)
}

func (logger *Logger) Panic(args ...interface{}) {
	entry.Log(PanicLevel, args...)
	panic(fmt.Sprint(args...))
}

//
// Printf style
//

func (logger *Logger) Logf(level Level, format string, args ...interface{}) {
	// TODO
}

func (logger *Logger) Tracef(format string, args ...interface{}) {
	entry.Logf(TraceLevel, format, args...)
}

func (logger *Logger) Debugf(format string, args ...interface{}) {
	entry.Logf(DebugLevel, format, args...)
}

func (logger *Logger) Infof(format string, args ...interface{}) {
	entry.Logf(InfoLevel, format, args...)
}

func (logger *Logger) Printf(format string, args ...interface{}) {
	entry.Infof(format, args...)
}

func (logger *Logger) Warnf(format string, args ...interface{}) {
	entry.Logf(WarnLevel, format, args...)
}

func (logger *Logger) Warningf(format string, args ...interface{}) {
	entry.Warnf(format, args...)
}

func (logger *Logger) Errorf(format string, args ...interface{}) {
	entry.Logf(ErrorLevel, format, args...)
}

func (logger *Logger) Fatalf(format string, args ...interface{}) {
	entry.Logf(FatalLevel, format, args...)
	entry.Logger.Exit(1)
}

func (logger *Logger) Panicf(format string, args ...interface{}) {
	entry.Logf(PanicLevel, format, args...)
}

//
// Println style
//

func (logger *Logger) Logln(level Level, args ...interface{}) {
	// TODO
}

func (logger *Logger) Traceln(args ...interface{}) {
	entry.Logln(TraceLevel, args...)
}

func (logger *Logger) Debugln(args ...interface{}) {
	entry.Logln(DebugLevel, args...)
}

func (logger *Logger) Infoln(args ...interface{}) {
	entry.Logln(InfoLevel, args...)
}

func (logger *Logger) Println(args ...interface{}) {
	entry.Infoln(args...)
}

func (logger *Logger) Warnln(args ...interface{}) {
	entry.Logln(WarnLevel, args...)
}

func (logger *Logger) Warningln(args ...interface{}) {
	entry.Warnln(args...)
}

func (logger *Logger) Errorln(args ...interface{}) {
	entry.Logln(ErrorLevel, args...)
}

func (logger *Logger) Fatalln(args ...interface{}) {
	entry.Logln(FatalLevel, args...)
	entry.Logger.Exit(1)
}

func (logger *Logger) Panicln(args ...interface{}) {
	entry.Logln(PanicLevel, args...)
}
