package otelzap

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogger_Sugar(t *testing.T) {
	// Setup
	baseLogger, _ := zap.NewProduction()
	logger := &Logger{Logger: baseLogger}

	// Test
	sugared := logger.Sugar()
	assert.NotNil(t, sugared)
	assert.IsType(t, &SugaredLogger{}, sugared)
}

func TestLogger_With(t *testing.T) {
	// Setup
	baseLogger, _ := zap.NewProduction()
	logger := &Logger{Logger: baseLogger}

	// Test cases
	fields := []zapcore.Field{
		zap.String("key1", "value1"),
		zap.Int("key2", 123),
	}

	withLogger := logger.With(fields...)
	assert.NotNil(t, withLogger)
	assert.IsType(t, &Logger{}, withLogger)
}

func TestLogger_Ctx(t *testing.T) {
	// Setup
	baseLogger, _ := zap.NewProduction()
	logger := &Logger{Logger: baseLogger}
	oldTracerProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	t.Cleanup(func() {
		otel.SetTracerProvider(oldTracerProvider)
	})

	tests := []struct {
		name      string
		createCtx func() context.Context
		shouldAdd bool
	}{
		{
			name: "With valid span context",
			createCtx: func() context.Context {
				tracer := otel.Tracer("")
				ctx := context.Background()
				ctx, _ = tracer.Start(ctx, "test")
				return ctx
			},
			shouldAdd: true,
		},
		{
			name: "With invalid span context",
			createCtx: func() context.Context {
				return context.Background()
			},
			shouldAdd: false,
		},
		{
			name: "With nil context",
			createCtx: func() context.Context {
				return nil
			},
			shouldAdd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.createCtx()
			resultLogger := logger.Ctx(ctx)

			assert.NotNil(t, resultLogger)
			assert.IsType(t, &Logger{}, resultLogger)

			if tt.shouldAdd {
				assert.NotEqual(t, logger, resultLogger)
			} else {
				assert.Equal(t, logger, resultLogger)
			}
		})
	}
}

func TestSugaredLogger_Ctx(t *testing.T) {
	// Setup
	baseLogger, _ := zap.NewProduction()
	sugared := &SugaredLogger{SugaredLogger: baseLogger.Sugar()}
	oldTracerProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	t.Cleanup(func() {
		otel.SetTracerProvider(oldTracerProvider)
	})

	tests := []struct {
		name      string
		createCtx func() context.Context
		shouldAdd bool
	}{
		{
			name: "With valid span context",
			createCtx: func() context.Context {
				tracer := otel.Tracer("")
				ctx := context.Background()
				ctx, _ = tracer.Start(ctx, "test")
				return ctx
			},
			shouldAdd: true,
		},
		{
			name: "With invalid span context",
			createCtx: func() context.Context {
				return context.Background()
			},
			shouldAdd: false,
		},
		{
			name: "With nil context",
			createCtx: func() context.Context {
				return nil
			},
			shouldAdd: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.createCtx()
			resultLogger := sugared.Ctx(ctx)

			assert.NotNil(t, resultLogger)
			assert.IsType(t, &SugaredLogger{}, resultLogger)

			if tt.shouldAdd {
				// When we have a valid span context, we expect a new instance with different fields
				assert.NotEqual(t, sugared.SugaredLogger, resultLogger.SugaredLogger)
			} else {
				// When no span context, we expect a new instance but with the same underlying logger
				assert.Equal(t, sugared.SugaredLogger, resultLogger.SugaredLogger)
			}
		})
	}
}

func TestLogger_Integration(t *testing.T) {
	// Setup
	baseLogger, _ := zap.NewProduction()
	logger := &Logger{Logger: baseLogger}

	// Test chaining
	ctx := context.Background()

	// Logger -> With -> Ctx -> Sugar
	result1 := logger.With(zap.String("key", "value")).Ctx(ctx).Sugar()
	assert.NotNil(t, result1)
	assert.IsType(t, &SugaredLogger{}, result1)

	// Logger -> Sugar -> Ctx
	result2 := logger.Sugar().Ctx(ctx)
	assert.NotNil(t, result2)
	assert.IsType(t, &SugaredLogger{}, result2)

	// Logger -> Ctx -> With -> Sugar
	result3 := logger.Ctx(ctx).With(zap.String("key", "value")).Sugar()
	assert.NotNil(t, result3)
	assert.IsType(t, &SugaredLogger{}, result3)
}
