/*
Package telemetry provides tools and utilities for instrument tracing and logs for components,
using OpenTelemetry.

Features:
  - log: Provides an instrumented logger based on uber-go/zap logger that is fully integrated
    to OpenTelemetry for log exporting and standardization.
  - tracing: Provides an instrumented tracer for creating a managing spans and traces, fully
    integrated to OpenTelemetry.
  - metric: Provides OpenTelemetry metrics implementation with counters, gauges, and histograms.

Required Environment Variables:

	OTEL_SERVICE_NAME                 your-service-name
	OTEL_EXPORTER_OTLP_ENDPOINT       https://central-otel-collector-dev.qa.cobre.co
	OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT 4095

Basic Usage:

	cleanup, err := telemetry.Init(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to initialize telemetry: %v", err))
	}
	// The cleanup function is for calling it when the component is
	// about to shutdown to clean up resources and flush the data.
	defer cleanup()

Note: This package serves as an entrypoint for the log and tracing packages.
Its primary purpose is to provide a simple way to initialize all the required
OpenTelemetry providers before using them.

Author: Nicolás Minelli <nicolash@cobre.co>
*/
package telemetry
