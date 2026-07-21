# Tracing Package

![technology Go](https://img.shields.io/badge/go-1.24+-blue.svg)

The `tracing` package provides a simplified interface for distributed tracing using OpenTelemetry. It offers wrapper functions for common tracing operations like creating spans, recording errors, and adding attributes.

## Features

- **Error Recording**: Automatic error recording in spans with status setting
- **Attribute Management**: Simple span attribute addition
- **OpenTelemetry Integration**: Full compatibility with OpenTelemetry's trace API
- **Context Propagation**: Seamless trace context propagation across service boundaries

## Installation

```bash
go get github.com/MFN-AISystems/go-toolkit/telemetry/tracing
```

## Usage

### Adding Attributes

```go
import "go.opentelemetry.io/otel/attribute"

// Add an attribute to the current span
tracing.AddAttribute(ctx, attribute.String("key", "value"))
```

### Flushing Data

In case you need to flush the data before the application exits, you can use the `ForceFlush` method.

```go
// Flush the data
err := tracing.ForceFlush(ctx)
```

This method is resource-expensive and should be used with caution.

### Console Traces

By default, when telemetry is enabled, traces are exported to the production exporter configured
(OLTP by default).

If you want to enable console traces for debugging purposes, 
you can set the following environment variable:
```bash
OTEL_TRACES_STDOUT_ENABLED=true
```

This appends a new console exporter to the existing configured exporters, 
so you can have both console and production exporters enabled at the same time.

#### Note

This package is part of the larger [telemetry](../) module. Make sure to initialize the telemetry providers using the main `telemetry.Init()` function before using this package. See the main telemetry package documentation for initialization and configuration details.