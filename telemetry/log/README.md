# Logging Package

![technology Go](https://img.shields.io/badge/go-1.24+-blue.svg)

The `log` package provides a standardized logging interface using Zap logger. It offers structured logging capabilities with consistent field naming and log levels.

## Features

- **Context-Aware Logging**: Seamlessly integrate logging with Go contexts
- **OpenTelemetry Integration**: Automatic trace ID injection and correlation
- **Structured Logging**: JSON-formatted logs with consistent field naming
- **Multiple Log Levels**: Support for Debug, Info, Warn, Error levels
- **Field Management**: Easy addition of structured fields to log entries

## Installation

```bash
go get github.com/nminelli/go-toolkit/telemetry/log
```

## Usage

### Basic Logging

```go
import "github.com/nminelli/go-toolkit/telemetry/log"

// Create a logger
logger := log.NewLogger()

// Log messages at different levels
logger.Info("processing request", "requestID", "123")
logger.Error("operation failed", "error", err)
```

### Logging in Context

```go
// First you need to create the logger and inject it into the context
ctx := context.Background()
logger := log.NewLogger()
ctx = log.Context(ctx, logger)

// Now you can log messages with the logger in the context without the
// of having the logger reference
log.Info(ctx, "operation completed")
```

### Log with Fields

```go
// You can add fields to the log message
logger.Info("processing request", log.String("requestID", "123"))

// Also you can make the same using the context
log.Info(ctx, "processing request", log.String("requestID", "123"))
```

### Adding Fields to the Logger

```go
// You can add fields for all the logs of the logger, creating a new logger
// with the wanted fields.
logger = logger.With(log.String("component", "my-component"))
// Now all the logs will have the field `component` with the value `my-component`

// Also you can add fields to the context
ctx = log.With(ctx, log.String("component", "my-component"))
// Now all the logs in the context will have the field `component` with the value `my-component`
```

### Console Logs

By default, when telemetry is enabled, logs are exported to the production exporter configured
(OLTP by default).

If you want to enable logs in console for debugging purposes,
you can set the following environment variable:
```bash
OTEL_LOGS_STDOUT_ENABLED=true
```

This appends a new console exporter to the existing configured exporters,
so you can have both console and production exporters enabled at the same time.

#### Note

This package is part of the larger [telemetry](../) module. Make sure to initialize the telemetry providers using the main `telemetry.Init()` function before using this package. See the main telemetry package documentation for initialization and configuration details.