// Package log uses ZAP fmt, which is a small wrapper around [Uber log package](https://godoc.org/go.uber.org/zap).
package log

import (
	"go.uber.org/zap/zapcore"
)

// ParseLevel parses a Level based on a lowercase or all-caps ASCII
// representation of the log level. If the provided ASCII representation is
// invalid then returns a Level with InfoLevel.
//
// This is particularly useful when dealing with text input to configure log
// levels.
func ParseLevel(level string) zapcore.Level {
	l, err := zapcore.ParseLevel(level)
	if err != nil {
		return zapcore.InfoLevel
	}

	return l
}
