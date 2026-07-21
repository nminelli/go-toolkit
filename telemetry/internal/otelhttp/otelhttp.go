package otelhttp

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/semconv/v1.37.0/httpconv"
	"go.opentelemetry.io/otel/trace"

	"github.com/nminelli/go-toolkit/telemetry/tracing"
)

// OTelRoundTripper is an http.RoundTripper that instruments HTTP requests with OpenTelemetry.
// It wraps an existing RoundTripper and adds distributed tracing and metrics collection.
type OTelRoundTripper struct {
	base           http.RoundTripper
	tracer         trace.Tracer
	durationMetric httpconv.ClientRequestDuration
	config         *SpanNameConfig
}

// NewOTelRoundTripper creates a new OTelRoundTripper that wraps the provided base RoundTripper.
// If base is nil, http.DefaultTransport will be used.
// The optional config parameter allows customization of span name generation.
func NewOTelRoundTripper(base http.RoundTripper, instrumentName string, instrumentVersion string, config *SpanNameConfig) *OTelRoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	// This is because the current telemetry/tracing always return a NoOp tracer for testing, and
	// we cannot test this behavior. Until that's fixed, we will implement this approach.
	tracer := otel.Tracer(instrumentName, trace.WithInstrumentationVersion(instrumentVersion))
	meter := otel.Meter(instrumentName, metric.WithInstrumentationVersion(instrumentVersion))

	duration, err := httpconv.NewClientRequestDuration(
		meter,
		otelmetric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10),
	)
	if err != nil {
		panic(errors.New("failed to create http client duration metric: " + err.Error()))
	}

	return &OTelRoundTripper{
		base:           base,
		tracer:         tracer,
		durationMetric: duration,
		config:         config,
	}
}

// RoundTrip implements the http.RoundTripper interface.
// It instruments the HTTP request with OpenTelemetry tracing and metrics.
func (t *OTelRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	start := time.Now()
	spanName := SanitizeSpanName(req.Method, req.URL.Path, t.config)

	ctx, span := t.tracer.Start(ctx, spanName, trace.WithTimestamp(start), trace.WithSpanKind(trace.SpanKindClient))
	tracing.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	// Clone request with new context
	req = req.Clone(ctx)

	// Execute the request
	resp, err := t.base.RoundTrip(req)
	end := time.Now()
	defer span.End(trace.WithTimestamp(end))

	if resp != nil {
		t.recordTelemetry(ctx, req, resp, err, start, end)
	}

	return resp, err
}

// recordTelemetry records OpenTelemetry span attributes and metrics for the HTTP request.
func (t *OTelRoundTripper) recordTelemetry(ctx context.Context, req *http.Request, resp *http.Response, err error, start time.Time, end time.Time) {
	span := tracing.SpanFromContext(ctx)
	serverAddress := req.URL.Hostname()
	port, _ := strconv.Atoi(req.URL.Port())

	// Set span attributes according to OpenTelemetry semantic conventions
	span.SetAttributes(
		semconv.HTTPRequestMethodKey.String(req.Method),
		semconv.ServerAddress(serverAddress),
		semconv.ServerPortKey.Int(port),
		semconv.URLFull(req.URL.String()),
		semconv.HTTPResponseStatusCode(resp.StatusCode),
	)

	// Record errors only for 5xx status codes or if there was an error executing the request
	switch {
	case err != nil:
		tracing.RecordError(ctx, err)
	case resp.StatusCode >= 500:
		err = fmt.Errorf("%s", http.StatusText(resp.StatusCode))
		tracing.RecordError(ctx, err)
	}

	// Record duration metric
	attrs := []attribute.KeyValue{
		semconv.HTTPResponseStatusCode(resp.StatusCode),
	}

	if err != nil {
		attrs = append(attrs, semconv.ErrorType(err))
	}

	t.durationMetric.Record(ctx, end.Sub(start).Seconds(), httpconv.RequestMethodAttr(req.Method), serverAddress, port, attrs...)
}
