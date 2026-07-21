# Telemetry Package

![technology Go](https://img.shields.io/badge/go-1.24+-blue.svg)

The `telemetry` package provides tools and utilities for instrument tracing and logs for components,
using OpenTelemetry.

## Features

- **Logging**: Instrumented logger based on [uber-go/zap](https://github.com/uber-go/zap) fully integrated
  with OpenTelemetry for log exporting and standardization.
- **Tracing**: Instrumented tracer for creating and managing spans and traces, fully
  integrated with OpenTelemetry.
- **Metrics**: OpenTelemetry metrics implementation with counters, gauges, and histograms.

## Installation

```bash
go get github.com/nminelli/go-toolkit/telemetry
```

## Requirements

To use this package, you must have the following environment variables set:

| Name                                | Value                                            |
|-------------------------------------|--------------------------------------------------|
| `OTEL_SERVICE_NAME`                 | `your-service-name`                              |
| `OTEL_EXPORTER_OTLP_ENDPOINT`       | `https://central-otel-collector-dev.qa.cobre.co` |
| `OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT` | `4095`                                           |

#### OLTP Endpoint

Depending on the environment you are running your application, you should set the
`OTEL_EXPORTER_OTLP_ENDPOINT` variable to the corresponding value:

| env           | endpoint                                         |
|---------------|--------------------------------------------------|
| `local`-dev` | `https://central-otel-collector-dev.qa.cobre.co` |
| `qa`          | `https://central-otel-collector.qa.cobre.co`     |
| `production`  | `https://central-otel-collector.cobre.co`        |

## Usage

This package is an entrypoint for the above-mentioned packages. It's only purpose is to provide a
simple way to initialize all the required OpenTelemetry providers before using them.

So the first thing you need to do is, initialize the module:

```go
import (
	"context"
	"fmt"
	"github.com/nminelli/go-toolkit/telemetry"
)

func main() {
	cleanup, err := telemetry.Init(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to initialize telemetry: %v", err))
	}
	// The cleanup function is for calling it when the component is 
	// about to shutdown to clean up resources and flush the data.
	defer cleanup()
	
	// Your code here
}
```

### HTTP Client Instrumentation

The package provides a convenient way to instrument your HTTP clients with OpenTelemetry tracing
and metrics through the `NewOTelRoundTripper` function. This function creates an `http.RoundTripper`
that automatically:

- Creates spans for each HTTP request with appropriate semantic conventions
- Propagates trace context via HTTP headers (W3C Trace Context)
- Records HTTP client metrics (request duration with histogram buckets)
- Captures error information for failed requests (5xx status codes or errors)

#### Basic Usage

```go
import (
	"net/http"
	"github.com/nminelli/go-toolkit/telemetry"
)

// Initialize telemetry first
cleanup, err := telemetry.Init(context.Background())
if err != nil {
	panic(err)
}
defer cleanup()

// Create an instrumented HTTP client with default configuration
transport := telemetry.NewOTelRoundTripper(nil, "", "")
client := &http.Client{Transport: transport}

// Use the client as normal - all requests will be automatically traced
resp, err := client.Get("https://api.example.com/data")
```

#### Custom Tracer Name and Version

You can specify a custom tracer name and version to better identify your HTTP client in traces:

```go
// Create an instrumented HTTP client with custom tracer
transport := telemetry.NewOTelRoundTripper(
	nil,                              // base RoundTripper (nil = use http.DefaultTransport)
	"my-service/http-client",         // custom tracer name
	"v1.2.3",                         // tracer version
)
client := &http.Client{Transport: transport}
```

#### Chaining with Custom Transports

You can chain the OTel RoundTripper with your own custom transports:

```go
// Create a custom transport
customTransport := &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:     90 * time.Second,
}

// Wrap it with OTel instrumentation
transport := telemetry.NewOTelRoundTripper(customTransport, "my-service", "1.0.0")
client := &http.Client{Transport: transport}
```

#### What Gets Captured

The instrumented HTTP client captures:

**Trace Spans:**
- Span name: `{METHOD} {PATH}` (e.g., "GET /api/users")
- Span kind: Client
- Attributes:
  - `http.request.method`: HTTP method
  - `server.address`: Target server address
  - `server.port`: Target server port
  - `url.full`: Full request URL
  - `http.response.status_code`: Response status code

**Metrics:**
- `http.client.request.duration`: Histogram of request durations in seconds
- Labels include: HTTP method, status code, server address, port, and error type (if any)

**Error Recording:**
- Errors are recorded in the span for:
  - Network/transport errors
  - HTTP 5xx status codes

### Testing

To run the tests for this package, use the following command:

```shell
go test ./...
```

# Contribution

**Authors:**

- Nicolas Minelli <minellinh@gmail.com>