package otelzap

import (
	otel "github.com/agoda-com/opentelemetry-logs-go/logs"
	"go.uber.org/zap/zapcore"
)

// NewOtelCore creates new OpenTelemetry Core to export logs in OTLP format
func NewOtelCore(loggerProvider otel.LoggerProvider, opts ...Option) zapcore.Core {
	logger := loggerProvider.Logger(
		instrumentationScope.Name,
		otel.WithInstrumentationVersion(instrumentationScope.Version),
	)

	c := &otlpCore{
		logger:       logger,
		levelEnabler: zapcore.InfoLevel,
	}
	for _, apply := range opts {
		apply(c)
	}

	return c
}

// Option is a function that applies an option to an OpenTelemetry Core
type Option func(c *otlpCore)

// WithLevel sets the minimum level for the OpenTelemetry Core log to be exported
func WithLevel(level zapcore.Level) Option {
	return func(c *otlpCore) {
		c.levelEnabler = level
	}
}

// WithLevelEnabler sets the zapcore.LevelEnabler for determining which log levels to export
func WithLevelEnabler(levelEnabler zapcore.LevelEnabler) Option {
	return func(c *otlpCore) {
		c.levelEnabler = levelEnabler
	}
}
