package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestParsesValidLevel(t *testing.T) {
	assert.Equal(t, zapcore.DebugLevel, ParseLevel("debug"))
	assert.Equal(t, zapcore.InfoLevel, ParseLevel("info"))
	assert.Equal(t, zapcore.WarnLevel, ParseLevel("warn"))
	assert.Equal(t, zapcore.ErrorLevel, ParseLevel("error"))
	assert.Equal(t, zapcore.DPanicLevel, ParseLevel("dpanic"))
	assert.Equal(t, zapcore.PanicLevel, ParseLevel("panic"))
	assert.Equal(t, zapcore.FatalLevel, ParseLevel("fatal"))
}

func TestParsesInvalidLevelAsInfo(t *testing.T) {
	assert.Equal(t, zapcore.InfoLevel, ParseLevel("invalid"))
	assert.Equal(t, zapcore.InfoLevel, ParseLevel(""))
	assert.Equal(t, zapcore.InfoLevel, ParseLevel("123"))
}
