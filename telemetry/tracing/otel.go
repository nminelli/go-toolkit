package tracing

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var once sync.Once

// Init initializes the tracing provider with the environment configuration.
// Returns an error if initialization fails.
func Init(ctx context.Context, res *resource.Resource) (err error) {
	once.Do(func() {
		err = initTracerProvider(ctx, res)
		if err != nil {
			err = fmt.Errorf("failed to initialize tracer provider: %v", err)
		}
	})

	return err
}

// Shutdown stops the tracing provider and flushes any remaining data.
// After shutdown the provider will function as a NoOp provider.
func Shutdown(ctx context.Context) error {
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok || tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}

// ForceFlush flushes any remaining traces in the provider.
// This is not a performant operation and should be used sparingly (like in tests or lambda
// functions).
func ForceFlush(ctx context.Context) error {
	tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok || tp == nil {
		return nil
	}
	return tp.ForceFlush(ctx)
}

// TP returns the tracer provider.
// If the tracer provider is not initialized and testing is enabled, a NoOp provider is returned.
func TP() trace.TracerProvider {
	return otel.GetTracerProvider()
}

// initTracerProvider creates a new TracerProvider instance.
// It initializes a serverless oriented telemetry client with the environment configuration.
// Returns the TracerProvider instance or an error if initialization fails.
func initTracerProvider(ctx context.Context, res *resource.Resource) error {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	if os.Getenv("OTEL_SERVICE_NAME") == "" {
		// Telemetry is disabled.
		return nil
	}

	var exp sdktrace.SpanExporter
	var err error

	switch protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); protocol {
	case "grpc":
		exp, err = otlptracegrpc.New(ctx)
	default:
		exp, err = otlptracehttp.New(ctx)
	}

	if err != nil {
		return err
	}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	}

	if os.Getenv("OTEL_TRACES_STDOUT_ENABLED") == "true" {
		stdoutExp, err := stdouttrace.New()
		if err != nil {
			fmt.Printf("failed to initialize stdouttrace: %v\n", err)
		} else {
			tpOpts = append(tpOpts, sdktrace.WithSyncer(stdoutExp))
		}
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)

	otel.SetTracerProvider(tp)

	return nil
}
