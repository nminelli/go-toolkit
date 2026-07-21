// context_test.go
package log

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/MFN-AISystems/go-toolkit/telemetry/log/otelzap"
)

func TestContext_WithLogger(t *testing.T) {
	// Create a logger
	logger := NewLogger()
	ctx := context.Background()

	// Add logger to context
	ctxWithLogger := Context(ctx, logger)
	require.NotNil(t, ctxWithLogger)

	// Retrieve logger from context
	retrievedLogger := FromContext(ctxWithLogger)
	assert.NotNil(t, retrievedLogger)
	assert.Same(t, logger, retrievedLogger)
}

func TestContext_WithoutLogger(t *testing.T) {
	ctx := context.Background()
	logger := FromContext(ctx)
	assert.NotNil(t, logger)
}

func TestWith_ExistingLogger(t *testing.T) {
	// Create initial logger with context
	baseLogger := NewLogger()
	ctx := Context(context.Background(), baseLogger)

	// Add fields
	fields := []Field{
		String("key1", "value1"),
		Int("key2", 42),
	}

	// Create new context with fields
	newCtx := With(ctx, fields...)

	// Retrieve and verify logger
	newLogger := FromContext(newCtx)
	require.NotNil(t, newLogger)
	assert.NotSame(t, baseLogger, newLogger)
}

func TestWith_NoLogger(t *testing.T) {
	ctx := context.Background()
	fields := []Field{String("key", "value")}

	// Add fields to context without existing logger
	newCtx := With(ctx, fields...)

	// Verify new logger was created
	newLogger := FromContext(newCtx)
	require.NotNil(t, newLogger)
	assert.IsType(t, &logger{}, newLogger)
}

func TestLoggingMethods_WithLogger(t *testing.T) {
	logger := NewLogger()
	ctx := Context(context.Background(), logger)
	fields := []Field{String("test_key", "test_value")}

	tests := []struct {
		name    string
		logFunc func(context.Context, string, ...Field)
	}{
		{"Debug", Debug},
		{"Info", Info},
		{"Warn", Warn},
		{"Error", Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				tt.logFunc(ctx, "test message", fields...)
			})
		})
	}
}

func TestLoggingMethods_WithoutLogger(t *testing.T) {
	ctx := context.Background()
	fields := []Field{String("test_key", "test_value")}

	tests := []struct {
		name    string
		logFunc func(context.Context, string, ...Field)
	}{
		{"Debug", Debug},
		{"Info", Info},
		{"Warn", Warn},
		{"Error", Error},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				tt.logFunc(ctx, "test message", fields...)
			})
		})
	}
}

func TestPanic_WithLogger(t *testing.T) {
	logger := NewLogger()
	ctx := Context(context.Background(), logger)

	assert.Panics(t, func() {
		Panic(ctx, "test panic message")
	})
}

func TestPanic_WithoutLogger(t *testing.T) {
	ctx := context.Background()

	assert.Panics(t, func() {
		Panic(ctx, "test panic message")
	})
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() context.Context
		expectedType interface{}
		verifyLogger func(*testing.T, Logger)
	}{
		{
			name: "with existing logger",
			setupContext: func() context.Context {
				return Context(context.Background(), NewLogger())
			},
			expectedType: &logger{},
			verifyLogger: func(t *testing.T, l Logger) {
				assert.NotNil(t, l)
				assert.IsType(t, &logger{}, l)
			},
		},
		{
			name: "without logger",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedType: &logger{},
			verifyLogger: func(t *testing.T, l Logger) {
				assert.NotNil(t, l)
				assert.IsType(t, &logger{}, l)
				assert.IsType(t, &otelzap.Logger{}, l.(*logger).Logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			logger := FromContext(ctx)
			tt.verifyLogger(t, logger)
		})
	}
}

func BenchmarkContext(b *testing.B) {
	logger := NewLogger()
	ctx := context.Background()
	fields := []Field{String("key", "value")}

	b.Run("Context/FromContext", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctxWithLogger := Context(ctx, logger)
			_ = FromContext(ctxWithLogger)
		}
	})

	b.Run("With", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = With(ctx, fields...)
		}
	})

	b.Run("Logging methods", func(b *testing.B) {
		ctxWithLogger := Context(ctx, logger)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				Info(ctxWithLogger, "test message", fields...)
			}
		})
	})
}

func TestGetTracingFields(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (context.Context, trace.Span)
		expected bool
	}{
		{
			name: "with valid span context",
			setup: func() (context.Context, trace.Span) {
				tracer := otel.Tracer("test")
				return tracer.Start(context.Background(), "test-span")
			},
			expected: true,
		},
		{
			name: "without span context",
			setup: func() (context.Context, trace.Span) {
				return context.Background(), nil
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldTracerProvider := otel.GetTracerProvider()
			otel.SetTracerProvider(sdktrace.NewTracerProvider())
			t.Cleanup(func() {
				otel.SetTracerProvider(oldTracerProvider)
			})

			ctx, span := tt.setup()
			if span != nil {
				defer span.End()
			}

			fields := getTracingFields(ctx)

			if !tt.expected {
				assert.Nil(t, fields)
			} else {
				assert.Len(t, fields, 2)
				assert.Equal(t, tagTraceID, fields[0].Key)
				assert.Equal(t, tagSpanID, fields[1].Key)
				assert.NotEmpty(t, fields[0].String)
				assert.NotEmpty(t, fields[1].String)
			}
		})
	}
}
