package log

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		opts    []Option
	}{
		{
			name: "default configuration",
			envVars: map[string]string{
				"LOG_LEVEL": "info",
			},
		},
		{
			name: "with service name",
			envVars: map[string]string{
				"LOG_LEVEL":         "debug",
				"OTEL_SERVICE_NAME": "test-service",
			},
		},
		{
			name: "with version",
			envVars: map[string]string{
				"LOG_LEVEL":   "info",
				"APP_VERSION": "1.0.0",
			},
		},
		{
			name: "with environment",
			envVars: map[string]string{
				"LOG_LEVEL":   "info",
				"ENVIRONMENT": "test",
			},
		},
		{
			name: "with all env vars",
			envVars: map[string]string{
				"LOG_LEVEL":         "debug",
				"OTEL_SERVICE_NAME": "test-service",
				"APP_VERSION":       "1.0.0",
				"ENVIRONMENT":       "test",
			},
		},
		{
			name: "with custom options",
			envVars: map[string]string{
				"LOG_LEVEL": "info",
			},
			opts: []Option{
				WithCaller(false),
				StacktraceOnError(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			originalEnv := make(map[string]string)
			for k := range tt.envVars {
				if v, ok := os.LookupEnv(k); ok {
					originalEnv[k] = v
				}
			}

			// Set test env vars
			for k, v := range tt.envVars {
				require.NoError(t, os.Setenv(k, v))
			}

			// Cleanup
			defer func() {
				for k := range tt.envVars {
					if v, ok := originalEnv[k]; ok {
						os.Setenv(k, v)
					} else {
						os.Unsetenv(k)
					}
				}
			}()

			// Create logger
			log := NewLogger(tt.opts...)
			require.NotNil(t, log)

			// Test logging
			log.Info("test message")
		})
	}
}

func TestLogger_With(t *testing.T) {
	// Create base logger
	baseLogger := NewLogger()

	tests := []struct {
		name   string
		fields []Field
	}{
		{
			name: "with single field",
			fields: []Field{
				String("key", "value"),
			},
		},
		{
			name: "with multiple fields",
			fields: []Field{
				String("key1", "value1"),
				Int("key2", 42),
				Bool("key3", true),
			},
		},
		{
			name:   "with no fields",
			fields: []Field{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			childLogger := baseLogger.With(tt.fields...)
			require.NotNil(t, childLogger)

			// Verify that child logger is a different instance
			assert.NotSame(t, baseLogger, childLogger)

			// Test logging with child logger
			childLogger.Info("test message")
		})
	}
}

func TestLoggerOptions(t *testing.T) {
	tests := []struct {
		name   string
		option Option
		verify func(*testing.T, *logConfig)
	}{
		{
			name:   "WithCaller true",
			option: WithCaller(true),
			verify: func(t *testing.T, cfg *logConfig) {
				assert.True(t, cfg.caller)
			},
		},
		{
			name:   "WithCaller false",
			option: WithCaller(false),
			verify: func(t *testing.T, cfg *logConfig) {
				assert.False(t, cfg.caller)
			},
		},
		{
			name:   "StacktraceOnError true",
			option: StacktraceOnError(true),
			verify: func(t *testing.T, cfg *logConfig) {
				assert.True(t, cfg.stacktrace)
			},
		},
		{
			name:   "StacktraceOnError false",
			option: StacktraceOnError(false),
			verify: func(t *testing.T, cfg *logConfig) {
				assert.False(t, cfg.stacktrace)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &logConfig{}
			tt.option(cfg)
			tt.verify(t, cfg)
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	cfg := &logConfig{}

	// Apply default options
	for _, opt := range _defaultOption {
		opt(cfg)
	}

	// Verify default values
	assert.True(t, cfg.stacktrace)
	assert.True(t, cfg.caller)
}

func TestLoggerInterface(t *testing.T) {
	var _ Logger = (*logger)(nil)
}

func TestLoggerMethods(t *testing.T) {
	// Set up environment
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")

	log := NewLogger()

	// Test all logging methods
	tests := []struct {
		name  string
		logFn func(string, ...Field)
	}{
		{"Debug", log.Debug},
		{"Info", log.Info},
		{"Warn", log.Warn},
		{"Error", log.Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.logFn("test message", String("key", "value"))
		})
	}
}

func TestLoggerWithContext(t *testing.T) {
	// Set up environment
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("LOG_LEVEL")

	ctx := context.Background()

	// Test context-aware logging methods
	tests := []struct {
		name  string
		logFn func(context.Context, string, ...Field)
	}{
		{"DebugContext", Debug},
		{"InfoContext", Info},
		{"WarnContext", Warn},
		{"ErrorContext", Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.logFn(ctx, "test message", String("key", "value"))
		})
	}
}

func TestLoggerWith_ChainedCalls(t *testing.T) {
	log := NewLogger()

	// Test chained With calls
	chainedLogger := log.
		With(String("key1", "value1")).
		With(String("key2", "value2"))

	require.NotNil(t, chainedLogger)
	chainedLogger.Info("test message")
}

func TestLoggerPanic(t *testing.T) {
	log := NewLogger()

	assert.Panics(t, func() {
		log.Panic("test panic message")
	})
}

func TestLoggerPanicContext(t *testing.T) {
	ctx := context.Background()

	assert.Panics(t, func() {
		Panic(ctx, "test panic message")
	})
}

func BenchmarkLogger(b *testing.B) {
	log := NewLogger()

	b.Run("Info with no fields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			log.Info("test message")
		}
	})

	b.Run("Info with fields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			log.Info("test message",
				String("key1", "value1"),
				Int("key2", 42),
			)
		}
	})

	b.Run("With + Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			log.With(String("key", "value")).
				Info("test message")
		}
	})
}
