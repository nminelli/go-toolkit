package log

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	otel "github.com/agoda-com/opentelemetry-logs-go"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/otlp/otlplogs"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/stdout/stdoutlogs"
	sdklogs "github.com/agoda-com/opentelemetry-logs-go/sdk/logs"
	"go.opentelemetry.io/otel/sdk/resource"
)

var (
	lp     *sdklogs.LoggerProvider
	noopLP = sdklogs.NewLoggerProvider()

	once sync.Once
)

// Init initializes the logger provider with the environment configuration.
// Returns an error if initialization fails.
func Init(ctx context.Context, res *resource.Resource) (err error) {
	once.Do(func() {
		lp, err = newLoggerProvider(ctx, res)
		if err != nil {
			err = fmt.Errorf("failed to initialize logger provider: %w", err)
		}
	})

	return err
}

// Shutdown stops the logger provider and flushes any remaining traces.
// After shutdown the provider will function as a NoOp provider.
func Shutdown(ctx context.Context) error {
	return lp.Shutdown(ctx)
}

// ForceFlush flushes any remaining logs in the provider.
// This is not a performant operation and should be used sparingly (like in tests or lambda
// functions).
func ForceFlush(ctx context.Context) error {
	return lp.ForceFlush(ctx)
}

// LP returns the logger provider.
// If the logger provider is not initialized and testing is enabled, a NoOp provider is returned.
func LP() *sdklogs.LoggerProvider {
	if lp == nil || testing.Testing() {
		return noopLP
	}

	return lp
}

// newLoggerProvider creates a new OpenTelemetry logger provider with
// the default configuration.
func newLoggerProvider(ctx context.Context, res *resource.Resource) (*sdklogs.LoggerProvider, error) {
	var exp sdklogs.LogRecordExporter
	var err error
	var exporterOption sdklogs.LoggerProviderOption

	telemetryEnabled := os.Getenv("OTEL_SERVICE_NAME") != ""
	if !telemetryEnabled {
		exp, err = stdoutlogs.NewExporter()
		if err != nil {
			return nil, err
		}
		exporterOption = sdklogs.WithSyncer(exp)
	} else {
		exp, err = otlplogs.NewExporter(ctx)
		if err != nil {
			return nil, err
		}
		exporterOption = sdklogs.WithBatcher(exp)
	}

	lpOpts := []sdklogs.LoggerProviderOption{
		exporterOption,
		sdklogs.WithResource(res),
	}
	if telemetryEnabled && os.Getenv("OTEL_LOGS_STDOUT_ENABLED") == "true" {
		stdoutExp, err := stdoutlogs.NewExporter()
		if err != nil {
			fmt.Printf("failed to initialize stdoutlog: %v\n", err)
		} else {
			lpOpts = append(lpOpts, sdklogs.WithSyncer(stdoutExp))
		}
	}

	lp = sdklogs.NewLoggerProvider(lpOpts...)

	otel.SetLoggerProvider(lp)

	return lp, nil
}
