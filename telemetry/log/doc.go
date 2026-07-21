/*
Package log provides a standardized logging interface using Zap logger with structured logging
capabilities and consistent field naming and log levels. It integrates with OpenTelemetry
for distributed tracing correlation.

Features:
  - Logging in Context
  - Integration with OpenTelemetry trace IDs
  - Structured logging with consistent field naming
  - Multiple log levels (Info, Error, Warn, Panic)

Basic Usage:

	logger := log.NewLogger()
	logger.Info("processing request", log.String("requestID", "123"))
	logger.Error("operation failed", log.String("error", err.Error()))

Logging in Context:

	ctx := context.Background()
	logger := log.NewLogger()
	ctx = log.Context(ctx, logger)
	log.Info(ctx, "operation completed")

Adding Fields:

	logger = logger.With(log.String("component", "my-component"))
	// or with context
	ctx = log.With(ctx, log.String("component", "my-component"))

Note: This package is part of the telemetry module. Initialize telemetry providers
using telemetry.Init() before using this package.
*/
package log
