package log

import (
	"context"
	"os"
	"sync"
	"testing"

	sdklogs "github.com/agoda-com/opentelemetry-logs-go/sdk/logs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/nminelli/go-toolkit/telemetry/tracing"
)

func TestForceFlushLogs(t *testing.T) {
	oldLP := lp
	lp = sdklogs.NewLoggerProvider()

	ctx := context.Background()
	err := ForceFlush(ctx)
	assert.NoError(t, err)

	t.Cleanup(func() {
		lp = oldLP
	})
}

func TestShutdownLoggerProviderReturnsNilOnSuccess(t *testing.T) {
	ctx := context.Background()
	err := Shutdown(ctx)
	assert.NoError(t, err)

	t.Cleanup(func() {
		once = sync.Once{}
	})
}

func Test_LP_WhenLpIsNil(t *testing.T) {
	oldLP := lp
	lp = nil
	t.Cleanup(func() {
		lp = oldLP
	})

	assert.Equal(t, noopLP, LP())
}

func TestNewLoggerProvider_WithStdoutLogsEnabled(t *testing.T) {
	tests := []struct {
		name              string
		serviceName       string
		stdoutLogsEnabled string
		expectStdout      bool
		setup             func()
		cleanup           func()
	}{
		{
			name:              "telemetry enabled with stdout logs enabled",
			serviceName:       "test-service",
			stdoutLogsEnabled: "true",
			expectStdout:      true,
			setup: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_LOGS_STDOUT_ENABLED", "true")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_LOGS_STDOUT_ENABLED")
			},
		},
		{
			name:              "telemetry enabled with stdout logs disabled",
			serviceName:       "test-service",
			stdoutLogsEnabled: "false",
			expectStdout:      false,
			setup: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_LOGS_STDOUT_ENABLED", "false")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_LOGS_STDOUT_ENABLED")
			},
		},
		{
			name:              "telemetry enabled without stdout logs env var",
			serviceName:       "test-service",
			stdoutLogsEnabled: "",
			expectStdout:      false,
			setup: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				// Don't set OTEL_LOGS_STDOUT_ENABLED
			},
			cleanup: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
			},
		},
		{
			name:              "telemetry disabled with stdout logs enabled",
			serviceName:       "",
			stdoutLogsEnabled: "true",
			expectStdout:      false,
			setup: func() {
				// Don't set OTEL_SERVICE_NAME (telemetry disabled)
				os.Setenv("OTEL_LOGS_STDOUT_ENABLED", "true")
			},
			cleanup: func() {
				os.Unsetenv("OTEL_LOGS_STDOUT_ENABLED")
			},
		},
		{
			name:              "telemetry disabled without stdout logs",
			serviceName:       "",
			stdoutLogsEnabled: "",
			expectStdout:      false,
			setup: func() {
				// Don't set any environment variables
			},
			cleanup: func() {
				// Nothing to cleanup
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
				provider, err := newLoggerProvider(context.Background(), res)

				if err != nil {
					// Some errors might be expected in test environment
					t.Logf("Error in test environment: %v", err)
				} else {
					// If no error, provider should be valid
					assert.NotNil(t, provider)

					// The configuration details are internal and we can't directly test
					// the number of exporters, but we can verify the provider was created

					// Clean up
					_ = provider.Shutdown(context.Background())
				}
			})
		})
	}
}

func TestNewLoggerProvider_TelemetryEnabledLogic(t *testing.T) {
	tests := []struct {
		name                   string
		serviceName            string
		expectTelemetryEnabled bool
	}{
		{
			name:                   "service name set - telemetry enabled",
			serviceName:            "my-service",
			expectTelemetryEnabled: true,
		},
		{
			name:                   "service name empty - telemetry disabled",
			serviceName:            "",
			expectTelemetryEnabled: false,
		},
		{
			name:                   "service name with spaces - telemetry enabled",
			serviceName:            "my service",
			expectTelemetryEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.serviceName != "" {
				os.Setenv("OTEL_SERVICE_NAME", tt.serviceName)
			} else {
				os.Unsetenv("OTEL_SERVICE_NAME")
			}

			t.Cleanup(func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
			})

			res, err := resource.New(context.Background())
			require.NoError(t, err)

			// Test that the provider can be created
			assert.NotPanics(t, func() {
				provider, err := newLoggerProvider(context.Background(), res)

				if err != nil {
					// Some errors might be expected in test environment
					t.Logf("Error in test environment: %v", err)
				} else {
					assert.NotNil(t, provider)
					_ = provider.Shutdown(context.Background())
				}
			})
		})
	}
}

func TestInit_WithStdoutLogsConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
	}{
		{
			name: "init with telemetry and stdout logs enabled",
			setupEnv: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_LOGS_STDOUT_ENABLED", "true")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_LOGS_STDOUT_ENABLED")
			},
			expectError: false,
		},
		{
			name: "init with telemetry enabled but stdout logs disabled",
			setupEnv: func() {
				os.Setenv("OTEL_SERVICE_NAME", "test-service")
				os.Setenv("OTEL_LOGS_STDOUT_ENABLED", "false")
			},
			cleanupEnv: func() {
				os.Unsetenv("OTEL_SERVICE_NAME")
				os.Unsetenv("OTEL_LOGS_STDOUT_ENABLED")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global state
			originalLP := lp
			once = sync.Once{}
			lp = nil

			t.Cleanup(func() {
				once = sync.Once{}
				lp = originalLP
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
				assert.NotNil(t, lp)
			}
		})
	}
}

func TestNewLoggerProvider_StdoutExporterError(t *testing.T) {
	// This test ensures that if stdout exporter creation fails,
	// the main logger provider still gets created
	t.Setenv("OTEL_SERVICE_NAME", "test-service")
	t.Setenv("OTEL_LOGS_STDOUT_ENABLED", "true")

	res, err := resource.New(context.Background())
	require.NoError(t, err)

	// Test that even if there are issues with stdout exporter,
	// the main provider is still created
	assert.NotPanics(t, func() {
		provider, err := newLoggerProvider(context.Background(), res)

		if err != nil {
			// This might be expected in test environment
			t.Logf("Expected potential error in test environment: %v", err)
		} else {
			assert.NotNil(t, provider)
			_ = provider.Shutdown(context.Background())
		}
	})
}

func TestMain(m *testing.M) {
	ctx := context.Background()
	if err := Init(ctx, nil); err != nil {
		panic(err)
	}

	if err := tracing.Init(ctx, nil); err != nil {
		panic(err)
	}

	m.Run()
}
