package metric

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var once sync.Once

// Init initializes the meter provider with the environment configuration.
// Returns an error if initialization fails.
func Init(ctx context.Context, res *resource.Resource) (err error) {
	once.Do(func() {
		err = initMeterProvider(ctx, res)
		if err != nil {
			err = fmt.Errorf("failed to initialize meter provider: %v", err)
		}

		if err = runtime.Start(); err != nil {
			err = fmt.Errorf("failed to start runtime instrumentation: %v", err)
		}
	})

	return err
}

// Shutdown stops the tracing provider and flushes any remaining data.
// After shutdown, the provider will function as a NoOp provider.
func Shutdown(ctx context.Context) error {
	mp, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider)
	if !ok || mp == nil {
		return nil
	}

	return mp.Shutdown(ctx)
}

// ForceFlush flushes any remaining traces in the provider.
// This is not a performant operation and should be used sparingly (like in tests or lambda
// functions).
func ForceFlush(ctx context.Context) error {
	mp, ok := otel.GetMeterProvider().(*sdkmetric.MeterProvider)
	if !ok || mp == nil {
		return nil
	}

	return mp.ForceFlush(ctx)
}

func initMeterProvider(ctx context.Context, res *resource.Resource) error {
	if os.Getenv("OTEL_SERVICE_NAME") == "" {
		// Telemetry is disabled.
		return nil
	}

	// For some reason the SDK doesn't allow setting these options via code, so we need to force it
	// via environment variables.
	_ = os.Setenv("OTEL_EXPORTER_OTLP_METRICS_DEFAULT_HISTOGRAM_AGGREGATION", "base2_exponential_bucket_histogram")
	_ = os.Setenv("OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE", "delta")

	var exp sdkmetric.Exporter
	var err error

	switch protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol {
	case "grpc":
		exp, err = otlpmetricgrpc.New(ctx)
	default:
		exp, err = otlpmetrichttp.New(ctx)
	}

	if err != nil {
		return err
	}

	mpOpts := []sdkmetric.Option{
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(5*time.Second)),
		),
		sdkmetric.WithResource(res),
	}

	if os.Getenv("OTEL_METRICS_STDOUT_ENABLED") == "true" {
		stdoutExp, err := stdoutmetric.New()
		if err != nil {
			fmt.Printf("failed to initialize stdoutmetric: %v\n", err)
		} else {
			mpOpts = append(mpOpts, sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(stdoutExp, sdkmetric.WithInterval(1*time.Minute))))
		}
	}

	mp := sdkmetric.NewMeterProvider(mpOpts...)

	otel.SetMeterProvider(mp)

	return nil
}
