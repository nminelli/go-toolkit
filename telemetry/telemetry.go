package telemetry

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/MFN-AISystems/go-toolkit/telemetry/internal/otelhttp"
	"github.com/MFN-AISystems/go-toolkit/telemetry/log"
	"github.com/MFN-AISystems/go-toolkit/telemetry/metric"
	"github.com/MFN-AISystems/go-toolkit/telemetry/tracing"
)

// Init initializes all the OpenTelemetry providers using the environment configuration.
// It returns a cleanup function that should be called to shut down the providers and an error
// if the initialization fails.
func Init(ctx context.Context) (func(), error) {
	ctx, cancel := context.WithCancel(ctx)
	res, err := newOtelResource(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := tracing.Init(ctx, res); err != nil {
		cancel()
		return nil, err
	}

	if err := log.Init(ctx, res); err != nil {
		cancel()
		return nil, err
	}

	if err := metric.Init(ctx, res); err != nil {
		cancel()
		return nil, err
	}

	cleanup := func() {
		cancel()

		// We give the providers 10 seconds to shut down gracefully.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := tracing.Shutdown(ctx); err != nil {
			log.Error(ctx, "failed to shutdown tracing provider", log.Err(err))
		}

		if err := log.Shutdown(ctx); err != nil {
			log.Error(ctx, "failed to shutdown log provider", log.Err(err))
		}

		if err := metric.Shutdown(ctx); err != nil {
			log.Error(ctx, "failed to shutdown metric provider", log.Err(err))
		}
	}

	return cleanup, nil
}

func newOtelResource(ctx context.Context) (*resource.Resource, error) {
	var attrs []attribute.KeyValue
	if version := os.Getenv("APP_VERSION"); version != "" {
		attrs = append(attrs, semconv.ServiceVersion(version))
	}
	if environment := os.Getenv("ENVIRONMENT"); environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(environment))
	}
	if instanceID := os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME"); instanceID != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(instanceID))
	}
	attrs = append(attrs,
		semconv.TelemetrySDKLanguageGo,
		semconv.TelemetrySDKName("opentelemetry"),
		semconv.TelemetrySDKVersion(sdk.Version()),
	)

	return resource.New(ctx, resource.WithAttributes(attrs...))
}

const (
	_defaultTracerName = "github.com/MFN-AISystems/go-toolkit/telemetry"
)

// PathPattern represents a regex pattern for matching and sanitizing URL paths in span names.
// Patterns are evaluated in the order they appear, with first match winning.
// This helps reduce cardinality in distributed tracing systems.
//
// Example:
//
//	pattern := PathPattern{
//	    Matcher:     regexp.MustCompile(`^/([^/]+)/_doc/[^/]+$`),
//	    Replacement: "/$1/_doc/{id}",
//	}
type PathPattern struct {
	// Matcher is the compiled regex used to match URL paths
	Matcher *regexp.Regexp
	// Replacement is the template string used to replace matched paths
	Replacement string
}

// Option is a functional option for configuring an OTelRoundTripper.
type Option func(*otelhttp.SpanNameConfig)

// WithAdditionalPatterns adds client-specific patterns that are evaluated after
// built-in heuristics. Patterns are evaluated in array order with first match winning.
//
// Example:
//
//	transport := telemetry.NewOTelRoundTripper(nil, "my-service", "v1.0.0",
//	    telemetry.WithAdditionalPatterns([]PathPattern{
//	        {
//	            Matcher:     regexp.MustCompile(`^/special/[^/]+$`),
//	            Replacement: "/special/{id}",
//	        },
//	    }),
//	)
func WithAdditionalPatterns(patterns []PathPattern) Option {
	return func(config *otelhttp.SpanNameConfig) {
		internalPatterns := make([]otelhttp.PathPattern, len(patterns))
		for i, p := range patterns {
			internalPatterns[i] = otelhttp.PathPattern{
				Matcher:     p.Matcher,
				Replacement: p.Replacement,
			}
		}
		config.AdditionalPatterns = internalPatterns
	}
}

// WithCustomSanitizer provides a custom function for generating span names.
// When set, this completely overrides the built-in heuristics and all pattern matching.
// Use this for complete control over span name generation for specific clients.
//
// Example:
//
//	transport := telemetry.NewOTelRoundTripper(nil, "legacy-service", "v1.0.0",
//	    telemetry.WithCustomSanitizer(func(method, path string) string {
//	        return method // Use method-only for all requests
//	    }),
//	)
func WithCustomSanitizer(sanitizer func(method, path string) string) Option {
	return func(config *otelhttp.SpanNameConfig) {
		config.CustomSanitizer = sanitizer
	}
}

// NewOTelRoundTripper creates a new OpenTelemetry-instrumented HTTP RoundTripper that adds
// distributed tracing and metrics collection to HTTP client requests.
//
// The returned RoundTripper wraps the provided base RoundTripper and automatically:
//   - Creates spans for each HTTP request with low-cardinality names (using built-in heuristics)
//   - Propagates trace context via HTTP headers
//   - Records HTTP client metrics (request duration)
//   - Captures error information for failed requests (5xx status codes or errors)
//
// Span names are automatically sanitized to reduce cardinality by replacing high-cardinality
// path segments (UUIDs, numeric IDs, hashes, tokens) with placeholders like {uuid}, {id}, etc.
// You can customize this behavior using the provided options.
//
// Parameters:
//   - base: The underlying http.RoundTripper to wrap. If nil, http.DefaultTransport is used.
//   - tracerName: The name to use for the OpenTelemetry tracer. If empty, uses the default
//     tracer name "github.com/MFN-AISystems/go-toolkit/telemetry".
//   - tracerVersion: The version to associate with the tracer. If tracerName is empty,
//     this parameter is ignored and the telemetry package version is used instead.
//   - opts: Optional functional options for customizing span name generation. See WithAdditionalPatterns
//     and WithCustomSanitizer for available options.
//
// Example:
//
//	// Create an instrumented HTTP client with default settings (built-in heuristics)
//	transport := telemetry.NewOTelRoundTripper(nil, "", "")
//	client := &http.Client{Transport: transport}
//
//	// Create an instrumented HTTP client with custom tracer
//	transport := telemetry.NewOTelRoundTripper(nil, "my-service/http-client", "1.2.3")
//	client := &http.Client{Transport: transport}
//
//	// Create an instrumented HTTP client with additional patterns
//	transport := telemetry.NewOTelRoundTripper(nil, "opensearch-client", "v1.0.0",
//	    telemetry.WithAdditionalPatterns([]PathPattern{
//	        {
//	            Matcher:     regexp.MustCompile(`^/custom-index/_doc/[^/]+$`),
//	            Replacement: "/custom-index/_doc/{id}",
//	        },
//	    }),
//	)
//	client := &http.Client{Transport: transport}
//
// The instrumented client will automatically create spans and record metrics for all
// HTTP requests made through it. Ensure that Init has been called before using this
// RoundTripper to properly configure the OpenTelemetry providers.
func NewOTelRoundTripper(base http.RoundTripper, tracerName string, tracerVersion string, opts ...Option) *otelhttp.OTelRoundTripper {
	if tracerName == "" {
		tracerName = _defaultTracerName
		tracerVersion = Version()
	}

	if tracerVersion == "" {
		tracerVersion = "0.0.0"
	}

	// Build config from options
	var config *otelhttp.SpanNameConfig
	if len(opts) > 0 {
		config = &otelhttp.SpanNameConfig{}
		for _, opt := range opts {
			opt(config)
		}
	}

	return otelhttp.NewOTelRoundTripper(base, tracerName, tracerVersion, config)
}
