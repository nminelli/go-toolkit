/*
Package tracing provides a simplified interface for distributed tracing using OpenTelemetry.
It offers wrapper functions for common tracing operations like creating spans, recording
errors, and adding attributes.

Features:
  - Error recording in spans with automatic status setting
  - Attribute management for spans
  - Integration with OpenTelemetry's trace API

Error Recording:

	// Record an error in the current span
	if err != nil {
	    tracing.RecordError(ctx, err)
	    // Or directly with span reference:
	    // span.RecordError(err)
	    // span.SetStatus(codes.Error, err.Error())
	}

Adding Attributes:

	// Add an attribute to the current span
	tracing.AddAttribute(ctx, attribute.String("key", "value"))

Data Flushing:

	// Force flush tracing data (use with caution, resource-expensive)
	err := tracing.ForceFlush(ctx)

Note: This package is part of the telemetry module. Initialize telemetry providers
using telemetry.Init() before using this package.
*/
package tracing
