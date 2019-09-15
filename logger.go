package logr

import (
	"fmt"
)

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

// Log checks that the level matches one or more targets, and
// if so, generates a log record that is added to the main
// queue (channel). Arguments are handled in the manner of fmt.Print.
func (logger *Logger) Log(level Level, args ...interface{}) {
	enabled, stacktrace := IsLevelEnabled(level)
	if enabled {
		rec := NewLogRec(level, logger, "", args, stacktrace)
		logr.in <- rec
	}
}

// Trace is a convenience method equivalent to `Log(TraceLevel, args...)`.
func (logger *Logger) Trace(args ...interface{}) {
	logger.Log(TraceLevel, args...)
}

// Debug is a convenience method equivalent to `Log(DebugLevel, args...)`.
func (logger *Logger) Debug(args ...interface{}) {
	logger.Log(DebugLevel, args...)
}

// Print ensures compatibility with std lib logger.
func (logger *Logger) Print(args ...interface{}) {
	logger.Info(args...)
}

// Info is a convenience method equivalent to `Log(InfoLevel, args...)`.
func (logger *Logger) Info(args ...interface{}) {
	logger.Log(InfoLevel, args...)
}

// Warn is a convenience method equivalent to `Log(WarnLevel, args...)`.
func (logger *Logger) Warn(args ...interface{}) {
	logger.Log(WarnLevel, args...)
}

// Error is a convenience method equivalent to `Log(ErrorLevel, args...)`.
func (logger *Logger) Error(args ...interface{}) {
	logger.Log(ErrorLevel, args...)
}

// Fatal is a convenience method equivalent to `Log(FatalLevel, args...)`
// followed by a call to os.Exit(1).
func (logger *Logger) Fatal(args ...interface{}) {
	logger.Log(FatalLevel, args...)
	Exit(1)
}

// Panic is a convenience method equivalent to `Log(PanicLevel, args...)`
// followed by a call to panic().
func (logger *Logger) Panic(args ...interface{}) {
	logger.Log(PanicLevel, args...)
	panic(fmt.Sprint(args...))
}

//
// Printf style
//

// Logf checks that the level matches one or more targets, and
// if so, generates a log record that is added to the main
// queue (channel). Arguments are handled in the manner of fmt.Printf.
func (logger *Logger) Logf(level Level, format string, args ...interface{}) {
	enabled, stacktrace := IsLevelEnabled(level)
	if enabled {
		rec := NewLogRec(level, logger, format, args, stacktrace)
		logr.in <- rec
	}
}

// Tracef is a convenience method equivalent to `Logf(TraceLevel, args...)`.
func (logger *Logger) Tracef(format string, args ...interface{}) {
	logger.Logf(TraceLevel, format, args...)
}

// Debugf is a convenience method equivalent to `Logf(DebugLevel, args...)`.
func (logger *Logger) Debugf(format string, args ...interface{}) {
	logger.Logf(DebugLevel, format, args...)
}

// Infof is a convenience method equivalent to `Logf(InfoLevel, args...)`.
func (logger *Logger) Infof(format string, args ...interface{}) {
	logger.Logf(InfoLevel, format, args...)
}

// Printf ensures compatibility with std lib logger.
func (logger *Logger) Printf(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

// Warnf is a convenience method equivalent to `Logf(WarnLevel, args...)`.
func (logger *Logger) Warnf(format string, args ...interface{}) {
	logger.Logf(WarnLevel, format, args...)
}

// Errorf is a convenience method equivalent to `Logf(ErrorLevel, args...)`.
func (logger *Logger) Errorf(format string, args ...interface{}) {
	logger.Logf(ErrorLevel, format, args...)
}

// Fatalf is a convenience method equivalent to `Logf(FatalLevel, args...)`
// followed by a call to os.Exit(1).
func (logger *Logger) Fatalf(format string, args ...interface{}) {
	logger.Logf(FatalLevel, format, args...)
	Exit(1)
}

// Panicf is a convenience method equivalent to `Logf(PanicLevel, args...)`
// followed by a call to panic().
func (logger *Logger) Panicf(format string, args ...interface{}) {
	logger.Logf(PanicLevel, format, args...)
}

//
// Println style
//

// Logln checks that the level matches one or more targets, and
// if so, generates a log record that is added to the main
// queue (channel). Arguments are handled in the manner of fmt.Println.
func (logger *Logger) Logln(level Level, args ...interface{}) {
	enabled, stacktrace := IsLevelEnabled(level)
	if enabled {
		rec := NewLogRec(level, logger, "", args, stacktrace)
		rec.newline = true
		logr.in <- rec
	}
}

// Traceln is a convenience method equivalent to `Logln(TraceLevel, args...)`.
func (logger *Logger) Traceln(args ...interface{}) {
	logger.Logln(TraceLevel, args...)
}

// Debugln is a convenience method equivalent to `Logln(DebugLevel, args...)`.
func (logger *Logger) Debugln(args ...interface{}) {
	logger.Logln(DebugLevel, args...)
}

// Infoln is a convenience method equivalent to `Logln(InfoLevel, args...)`.
func (logger *Logger) Infoln(args ...interface{}) {
	logger.Logln(InfoLevel, args...)
}

// Println ensures compatibility with std lib logger.
func (logger *Logger) Println(args ...interface{}) {
	logger.Infoln(args...)
}

// Warnln is a convenience method equivalent to `Logln(WarnLevel, args...)`.
func (logger *Logger) Warnln(args ...interface{}) {
	logger.Logln(WarnLevel, args...)
}

// Errorln is a convenience method equivalent to `Logln(ErrorLevel, args...)`.
func (logger *Logger) Errorln(args ...interface{}) {
	logger.Logln(ErrorLevel, args...)
}

// Fatalln is a convenience method equivalent to `Logln(FatalLevel, args...)`
// followed by a call to os.Exit(1).
func (logger *Logger) Fatalln(args ...interface{}) {
	logger.Logln(FatalLevel, args...)
	Exit(1)
}

// Panicln is a convenience method equivalent to `Logln(PanicLevel, args...)`
// followed by a call to panic().
func (logger *Logger) Panicln(args ...interface{}) {
	logger.Logln(PanicLevel, args...)
}
