// Package log uses ZAP fmt, which is a small wrapper around [Uber log package](https://godoc.org/go.uber.org/zap).
package log

import (
	"go.uber.org/zap/zapcore"
)

// Core is a minimal, fast logger interface. It's designed for library authors
// to wrap in a more user-friendly API.
type Core = zapcore.Core

// Logger is the interface that wraps methods needed for a valid logger implementation.
type Logger interface {
	// With creates a child logger and adds structured context to it. Fields added
	// to the child don't affect the parent, and vice versa.
	With(fields ...Field) Logger

	// DPanic logs a message at DPanicLevel. The message includes any fields
	// passed at the log site, as well as any fields accumulated on the logger.
	DPanic(msg string, fields ...Field)

	// Debug logs a message at DebugLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Debug(msg string, fields ...Field)

	// Error logs a message at ErrorLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Error(msg string, fields ...Field)

	// Info logs a message at InfoLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Info(msg string, fields ...Field)

	// Panic logs a message at PanicLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	//
	// The logger then panics, even if logging at PanicLevel is disabled.
	Panic(msg string, fields ...Field)

	// Fatal logs a message at FatalLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	//
	// The logger then calls os.Exit(1), even if logging at FatalLevel is disabled.
	// This is a hard stop, so no further log messages will be written.
	Fatal(msg string, fields ...Field)

	// Warn logs a message at WarnLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Warn(msg string, fields ...Field)
}
