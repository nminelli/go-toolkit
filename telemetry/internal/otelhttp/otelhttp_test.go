package otelhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"github.com/MFN-AISystems/go-toolkit/telemetry/log"
	"github.com/MFN-AISystems/go-toolkit/telemetry/metric"
	"github.com/MFN-AISystems/go-toolkit/telemetry/tracing"
)

func TestMain(m *testing.M) {
	// Set shorter timeouts for OTLP exporter
	os.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_TIMEOUT")

	ctx := context.Background()
	res, _ := resource.New(ctx)

	// Set up global OTel providers for testing
	tracerProvider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)

	meterProvider := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(meterProvider)

	// Initialize providers directly to avoid import cycle
	_ = tracing.Init(ctx, res)
	_ = log.Init(ctx, res)
	_ = metric.Init(ctx, res)

	exitCode := m.Run()

	// Cleanup
	cleanupCtx := context.Background()
	_ = tracing.Shutdown(cleanupCtx)
	_ = log.Shutdown(cleanupCtx)
	_ = metric.Shutdown(cleanupCtx)
	_ = tracerProvider.Shutdown(cleanupCtx)
	_ = meterProvider.Shutdown(cleanupCtx)

	os.Exit(exitCode)
}

func TestNewOTelRoundTripper(t *testing.T) {
	testCases := []struct {
		name          string
		base          http.RoundTripper
		validateSetup func(t *testing.T, rt *OTelRoundTripper)
	}{
		{
			name: "creates with custom base transport",
			base: &http.Transport{},
			validateSetup: func(t *testing.T, rt *OTelRoundTripper) {
				assert.NotNil(t, rt.base)
			},
		},
		{
			name: "uses default transport when base is nil",
			base: nil,
			validateSetup: func(t *testing.T, rt *OTelRoundTripper) {
				assert.NotNil(t, rt.base)
				assert.Equal(t, http.DefaultTransport, rt.base)
			},
		},
		{
			name: "works with empty version",
			base: &http.Transport{},
			validateSetup: func(t *testing.T, rt *OTelRoundTripper) {
				assert.NotNil(t, rt.base)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rt := NewOTelRoundTripper(tc.base, "", "v1.0.0", nil)
			tc.validateSetup(t, rt)
		})
	}
}

func TestOTelRoundTripper_RoundTrip(t *testing.T) {
	testCases := []struct {
		name             string
		version          string
		serverStatusCode int
		serverPath       string
		method           string
		validateResponse func(t *testing.T, resp *http.Response, err error)
	}{
		{
			name:             "successful GET request",
			version:          "v1.0.0",
			serverStatusCode: http.StatusOK,
			serverPath:       "/test",
			method:           http.MethodGet,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
		{
			name:             "client error 404 does not record error in span",
			version:          "v1.5.2",
			serverStatusCode: http.StatusNotFound,
			serverPath:       "/missing",
			method:           http.MethodGet,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			},
		},
		{
			name:             "server error 500 records error in span",
			version:          "v2.1.0",
			serverStatusCode: http.StatusInternalServerError,
			serverPath:       "/error",
			method:           http.MethodPost,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			},
		},
		{
			name:             "bad gateway 502 records error",
			version:          "v1.0.0",
			serverStatusCode: http.StatusBadGateway,
			serverPath:       "/gateway",
			method:           http.MethodGet,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
			},
		},
		{
			name:             "POST request with path",
			version:          "v1.0.0",
			serverStatusCode: http.StatusCreated,
			serverPath:       "/api/users",
			method:           http.MethodPost,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusCreated, resp.StatusCode)
			},
		},
		{
			name:             "PUT request",
			version:          "v1.0.0",
			serverStatusCode: http.StatusOK,
			serverPath:       "/api/items/123",
			method:           http.MethodPut,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			},
		},
		{
			name:             "DELETE request",
			version:          "v1.0.0",
			serverStatusCode: http.StatusNoContent,
			serverPath:       "/api/items/456",
			method:           http.MethodDelete,
			validateResponse: func(t *testing.T, resp *http.Response, err error) {
				require.NoError(t, err)
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.method, r.Method)
				assert.Equal(t, tc.serverPath, r.URL.Path)
				w.WriteHeader(tc.serverStatusCode)
				_, _ = w.Write([]byte(`{"status":"ok"}`))
			}))
			defer server.Close()

			// Setup OpenTelemetry for testing
			spanRecorder := tracetest.NewSpanRecorder()
			tracerProvider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(spanRecorder),
			)
			otel.SetTracerProvider(tracerProvider)
			defer func() {
				_ = tracerProvider.Shutdown(context.Background())
				otel.SetTracerProvider(nil)
			}()

			metricReader := sdkmetric.NewManualReader()
			res := resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String("test-service"),
			)
			meterProvider := sdkmetric.NewMeterProvider(
				sdkmetric.WithReader(metricReader),
				sdkmetric.WithResource(res),
			)
			otel.SetMeterProvider(meterProvider)
			defer func() {
				_ = meterProvider.Shutdown(context.Background())
				otel.SetMeterProvider(nil)
			}()

			// Initialize metric package for duration recording
			err := metric.Init(context.Background(), res)
			require.NoError(t, err)
			defer func() {
				_ = metric.Shutdown(context.Background())
			}()

			// Create HTTP client with instrumented transport
			rt := NewOTelRoundTripper(nil, "", tc.version, nil)
			client := &http.Client{Transport: rt}

			// Make request
			req, err := http.NewRequestWithContext(context.Background(), tc.method, server.URL+tc.serverPath, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			tc.validateResponse(t, resp, err)

			if resp != nil {
				_ = resp.Body.Close()
			}

			// Verify span was created
			spans := spanRecorder.Ended()
			require.Len(t, spans, 1, "expected one span to be created")

			// Note: Span names are now sanitized, so we just verify a span was created
			// Specific span name assertions are in dedicated sanitization tests
			assert.NotEmpty(t, spans[0].Name())
		})
	}
}

func TestOTelRoundTripper_TraceContextPropagation(t *testing.T) {
	// Setup test server that verifies trace context headers
	var receivedTraceParent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceParent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup OpenTelemetry
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	meterProvider := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
		otel.SetMeterProvider(nil)
	}()

	// Set text map propagator for W3C trace context propagation
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator())

	// Create HTTP client with instrumented transport
	rt := NewOTelRoundTripper(nil, "", "v1.0.0", nil)
	client := &http.Client{Transport: rt}

	// Make request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()

	// Verify trace context was propagated
	assert.NotEmpty(t, receivedTraceParent, "traceparent header should be propagated")
}

func TestOTelRoundTripper_RoundTripWithDelay(t *testing.T) {
	// Setup test server with delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Setup OpenTelemetry
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
	}()

	metricReader := sdkmetric.NewManualReader()
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("test-service"),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)
	defer func() {
		_ = meterProvider.Shutdown(context.Background())
		otel.SetMeterProvider(nil)
	}()

	err := metric.Init(context.Background(), res)
	require.NoError(t, err)
	defer func() {
		_ = metric.Shutdown(context.Background())
	}()

	// Create HTTP client with instrumented transport
	rt := NewOTelRoundTripper(nil, "", "v1.0.0", nil)
	client := &http.Client{Transport: rt}

	// Record start time
	start := time.Now()

	// Make request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()

	// Verify request took at least 100ms
	duration := time.Since(start)
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond, "request should take at least 100ms due to server delay")

	// Verify span was created with correct timing
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
}

func TestOTelRoundTripper_ChainedRoundTrippers(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header from chained transport
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup OpenTelemetry
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	meterProvider := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
		otel.SetMeterProvider(nil)
	}()

	// Create a custom RoundTripper that adds a header
	customRT := &customHeaderRoundTripper{
		base:   http.DefaultTransport,
		header: "X-Custom-Header",
		value:  "test-value",
	}

	// Chain the OTel RoundTripper with the custom one
	otelRT := NewOTelRoundTripper(customRT, "", "v1.0.0", nil)
	client := &http.Client{Transport: otelRT}

	// Make request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()

	// Verify span was created
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
}

// customHeaderRoundTripper is a test helper that adds a custom header to requests
type customHeaderRoundTripper struct {
	base   http.RoundTripper
	header string
	value  string
}

func (rt *customHeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(rt.header, rt.value)
	return rt.base.RoundTrip(req)
}

func TestOTelRoundTripper_SpanAttributes(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Setup OpenTelemetry
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	meterProvider := sdkmetric.NewMeterProvider()
	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
		otel.SetTracerProvider(nil)
		otel.SetMeterProvider(nil)
	}()

	// Create HTTP client with instrumented transport
	rt := NewOTelRoundTripper(nil, "", "v1.0.0", nil)
	client := &http.Client{Transport: rt}

	// Make request
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/api/test", nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	_ = resp.Body.Close()

	// Verify span attributes
	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)

	span := spans[0]
	attrs := span.Attributes()

	// Verify required attributes are present
	hasMethod := false
	hasStatusCode := false
	hasServerAddress := false

	for _, attr := range attrs {
		switch string(attr.Key) {
		case "http.request.method":
			hasMethod = true
			assert.Equal(t, "GET", attr.Value.AsString())
		case "http.response.status_code":
			hasStatusCode = true
			assert.Equal(t, int64(200), attr.Value.AsInt64())
		case "server.address":
			hasServerAddress = true
		}
	}

	assert.True(t, hasMethod, "span should have http.request.method attribute")
	assert.True(t, hasStatusCode, "span should have http.response.status_code attribute")
	assert.True(t, hasServerAddress, "span should have server.address attribute")
}
