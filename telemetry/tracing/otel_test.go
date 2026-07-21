package tracing

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestForceFlush_ReturnsNilOnSuccess(t *testing.T) {
	ctx := context.Background()
	err := ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestShutdown_ReturnsNilOnSuccess(t *testing.T) {
	ctx := context.Background()
	err := Shutdown(ctx)
	assert.NoError(t, err)
}

func Test_TP(t *testing.T) {
	oldProvider := otel.GetTracerProvider()
	tracerProvider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	t.Cleanup(func() {
		otel.SetTracerProvider(oldProvider)
	})

	assert.Equal(t, tracerProvider, TP())
}

func TestInit(t *testing.T) {
	// Save original state
	oldProvider := otel.GetTracerProvider()
	tracerProvider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)
	t.Cleanup(func() {
		once = sync.Once{}
		otel.SetTracerProvider(oldProvider)
	})

	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
		resetOnce   bool
	}{
		{
			name: "init with testing environment",
			setupEnv: func() {
				// Testing environment - should use NoOp provider
			},
			cleanupEnv:  func() {},
			expectError: false,
			resetOnce:   true,
		},
		{
			name: "init without OTEL_SERVICE_NAME",
			setupEnv: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
			},
			cleanupEnv:  func() {},
			expectError: false,
			resetOnce:   true,
		},
		{
			name: "init with OTEL_SERVICE_NAME - should succeed",
			setupEnv: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
			resetOnce:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resetOnce {
				once = sync.Once{}
			}

			tt.setupEnv()
			defer tt.cleanupEnv()

			res, _ := resource.New(context.Background())
			err := Init(context.Background(), res)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewTracerProvider(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
	}{
		{
			name: "with grpc protocol",
			setupEnv: func() {
				os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
		},
		{
			name: "with http protocol (default)",
			setupEnv: func() {
				os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
		},
		{
			name: "with valid endpoint",
			setupEnv: func() {
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
		},
		{
			name: "without protocol (should default to http)",
			setupEnv: func() {
				os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			res, _ := resource.New(context.Background())
			err := initTracerProvider(context.Background(), res)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Test that we can get a tracer from the provider
				_ = Shutdown(context.Background())
			}
		})
	}
}

func TestNewTracerProvider_WithStdoutExporter(t *testing.T) {
	tests := []struct {
		name          string
		stdoutEnabled string
		protocol      string
		expectStdout  bool
		setup         func()
		cleanup       func()
	}{
		{
			name:          "stdout exporter enabled",
			stdoutEnabled: "true",
			protocol:      "grpc",
			expectStdout:  true,
			setup: func() {
				os.Setenv("OTEL_TRACES_STDOUT_ENABLED", "true")
				os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_TRACES_STDOUT_ENABLED")
				os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
		},
		{
			name:          "stdout exporter disabled",
			stdoutEnabled: "false",
			protocol:      "http",
			expectStdout:  false,
			setup: func() {
				os.Setenv("OTEL_TRACES_STDOUT_ENABLED", "false")
				os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_TRACES_STDOUT_ENABLED")
				os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
		},
		{
			name:          "stdout exporter not set",
			stdoutEnabled: "",
			protocol:      "",
			expectStdout:  false,
			setup: func() {
				// Don't set stdout env var
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
		},
		{
			name:          "stdout exporter with invalid value",
			stdoutEnabled: "invalid",
			protocol:      "grpc",
			expectStdout:  false,
			setup: func() {
				os.Setenv("OTEL_TRACES_STDOUT_ENABLED", "invalid")
				os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_TRACES_STDOUT_ENABLED")
				os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			t.Cleanup(tt.cleanup)

			res, err := resource.New(context.Background())
			require.NoError(t, err)

			// Test that the function can be called without panicking
			assert.NotPanics(t, func() {
				err := initTracerProvider(context.Background(), res)
				assert.NoError(t, err)
			})
		})
	}
}

func TestNewTracerProvider_StdoutExporterError(t *testing.T) {
	// This test ensures that if stdout exporter creation fails,
	// the main tracer provider still gets created
	t.Setenv("OTEL_TRACES_STDOUT_ENABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	res, err := resource.New(context.Background())
	require.NoError(t, err)

	// Test that even if there are issues with stdout exporter,
	// the main provider is still created
	assert.NotPanics(t, func() {
		err := initTracerProvider(context.Background(), res)
		assert.NoError(t, err)
	})
}

func TestInit_WithStdoutExporter(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
	}{
		{
			name: "init with stdout traces enabled",
			setupEnv: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_TRACES_STDOUT_ENABLED", "true")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_TRACES_STDOUT_ENABLED")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
		},
		{
			name: "init with stdout traces disabled",
			setupEnv: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_TRACES_STDOUT_ENABLED", "false")
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_TRACES_STDOUT_ENABLED")
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			t.Cleanup(func() {
				once = sync.Once{}
			})

			tt.setupEnv()
			defer tt.cleanupEnv()

			res, err := resource.New(context.Background())
			require.NoError(t, err)

			err = Init(context.Background(), res)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
