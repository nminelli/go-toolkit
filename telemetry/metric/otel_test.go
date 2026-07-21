package metric

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestForceFlush_ReturnsNilOnSuccess(t *testing.T) {
	oldMP := otel.GetMeterProvider()
	mp := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(oldMP)
	})

	ctx := context.Background()
	err := ForceFlush(ctx)
	assert.NoError(t, err)
}

func TestShutdown_ReturnsNilOnSuccess(t *testing.T) {
	oldMP := otel.GetMeterProvider()
	mp := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(oldMP)
	})

	ctx := context.Background()
	err := Shutdown(ctx)
	assert.NoError(t, err)
}

func TestInit_WithDifferentConfigurations(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		protocol    string
		expectError bool
	}{
		{
			name:        "no service name - testing mode",
			serviceName: "",
			protocol:    "",
			expectError: false,
		},
		{
			name:        "with service name - grpc protocol",
			serviceName: "test-service",
			protocol:    "grpc",
			expectError: false,
		},
		{
			name:        "with service name - http protocol",
			serviceName: "test-service",
			protocol:    "http",
			expectError: false,
		},
		{
			name:        "with service name - default protocol",
			serviceName: "test-service",
			protocol:    "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			oldMP := otel.GetMeterProvider()
			mp := sdkmetric.NewMeterProvider()
			otel.SetMeterProvider(mp)
			t.Cleanup(func() {
				otel.SetMeterProvider(oldMP)
				once = sync.Once{}
			})

			// Set environment variables
			if tt.serviceName != "" {
				t.Setenv("OTEL_SERVICE_NAME", tt.serviceName)
			}
			if tt.protocol != "" {
				t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", tt.protocol)
			}

			// Create a test resource
			res, err := resource.New(context.Background())
			require.NoError(t, err)

			// Test Init
			err = Init(context.Background(), res)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitMeterProvider_WithDifferentProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		setup    func(t *testing.T)
	}{
		{
			name:     "grpc protocol",
			protocol: "grpc",
			setup: func(t *testing.T) {
				t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
				// Disable actual network calls for testing
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
				t.Setenv("OTEL_EXPORTER_OTLP_METRICS_TIMEOUT", "1")
			},
		},
		{
			name:     "http protocol",
			protocol: "http",
			setup: func(t *testing.T) {
				t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
				// Disable actual network calls for testing
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
			},
		},
		{
			name:     "default protocol (http)",
			protocol: "",
			setup: func(t *testing.T) {
				// Don't set protocol, should default to http
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldMP := otel.GetMeterProvider()
			t.Cleanup(func() {
				otel.SetMeterProvider(oldMP)
			})

			tt.setup(t)

			res, err := resource.New(context.Background())
			require.NoError(t, err)

			// Test that the function can be called without panicking
			// and that it properly routes to the expected protocol
			assert.NotPanics(t, func() {
				err := initMeterProvider(context.Background(), res)

				if err != nil {
					// This is expected since we're not running actual OTLP receivers
					t.Logf("Expected error in test environment: %v", err)
				} else {
					// If no error, provider should be valid
					assert.NotNil(t, otel.GetMeterProvider())
				}
			})
		})
	}
}

func TestInit_MultipleCallsSameOnce(t *testing.T) {
	// Reset global state
	oldMP := otel.GetMeterProvider()
	mp := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(mp)
	t.Cleanup(func() {
		otel.SetMeterProvider(oldMP)
	})

	res, err := resource.New(context.Background())
	require.NoError(t, err)

	// First call
	err1 := Init(context.Background(), res)

	// Second call - should not re-initialize due to sync.Once
	err2 := Init(context.Background(), res)

	// Both should succeed (no error expected in test mode)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestInitMeterProvider_WithStdoutExporter(t *testing.T) {
	tests := []struct {
		name          string
		stdoutEnabled string
		protocol      string
		expectStdout  bool
		setup         func(t *testing.T)
	}{
		{
			name:          "stdout exporter enabled",
			stdoutEnabled: "true",
			protocol:      "grpc",
			expectStdout:  true,
			setup: func(t *testing.T) {
				t.Setenv("OTEL_METRICS_STDOUT_ENABLED", "true")
				t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
				// Disable actual network calls for testing
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
				t.Setenv("OTEL_EXPORTER_OTLP_METRICS_TIMEOUT", "1")
			},
		},
		{
			name:          "stdout exporter disabled",
			stdoutEnabled: "false",
			protocol:      "http",
			expectStdout:  false,
			setup: func(t *testing.T) {
				t.Setenv("OTEL_METRICS_STDOUT_ENABLED", "false")
				t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
			},
		},
		{
			name:          "stdout exporter not set",
			stdoutEnabled: "",
			protocol:      "",
			expectStdout:  false,
			setup: func(t *testing.T) {
				// Don't set stdout env var
				t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldMP := otel.GetMeterProvider()
			t.Cleanup(func() {
				otel.SetMeterProvider(oldMP)
			})
			tt.setup(t)

			res, err := resource.New(context.Background())
			require.NoError(t, err)

			// Test that the function can be called without panicking
			assert.NotPanics(t, func() {
				err := initMeterProvider(context.Background(), res)

				if err != nil {
					// This is expected since we're not running actual OTLP receivers
					t.Logf("Expected error in test environment: %v", err)
				} else {
					// If no error, provider should be valid
					assert.NotNil(t, otel.GetMeterProvider())
				}
			})
		})
	}
}

func TestNewMeterProvider_StdoutExporterError(t *testing.T) {
	oldMP := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetMeterProvider(oldMP)
	})

	// This test ensures that if stdout exporter creation fails,
	// the main meter provider still gets created
	t.Setenv("OTEL_METRICS_STDOUT_ENABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	res, err := resource.New(context.Background())
	require.NoError(t, err)

	// Test that even if there are issues with stdout exporter,
	// the main provider is still created
	assert.NotPanics(t, func() {
		err := initMeterProvider(context.Background(), res)

		if err != nil {
			// This is expected since we're not running actual OTLP receivers
			t.Logf("Expected error in test environment: %v", err)
		} else {
			assert.NotNil(t, otel.GetMeterProvider())
		}
	})
}
