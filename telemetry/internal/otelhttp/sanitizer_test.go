package otelhttp

import (
	"context"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSanitizeSpanName_BuiltInHeuristics(t *testing.T) {
	defer setupMetricProvider(t)()

	testCases := []struct {
		name             string
		path             string
		method           string
		expectedSpanName string
	}{
		{
			name:             "UUID with hyphens",
			path:             "/users/550e8400-e29b-41d4-a716-446655440000",
			method:           http.MethodGet,
			expectedSpanName: "GET /users/{uuid}",
		},
		{
			name:             "UUID without hyphens",
			path:             "/files/550e8400e29b41d4a716446655440000",
			method:           http.MethodGet,
			expectedSpanName: "GET /files/{uuid}",
		},
		{
			name:             "OpenSearch document PUT",
			path:             "/my-index/_doc/doc_12345",
			method:           http.MethodPut,
			expectedSpanName: "PUT /my-index/_doc/{id}",
		},
		{
			name:             "OpenSearch documents SEARCH",
			path:             "/watchlist-entries-index/_search",
			method:           http.MethodPost,
			expectedSpanName: "POST /watchlist-entries-index/_search",
		},
		{
			name:             "OpenSearch document UPDATE",
			path:             "/logs-2024/_update/event_67890",
			method:           http.MethodPost,
			expectedSpanName: "POST /logs-2024/_update/{id}",
		},
		{
			name:             "OpenSearch document CREATE",
			path:             "/products/_create/prod_abc123",
			method:           http.MethodPut,
			expectedSpanName: "PUT /products/_create/{id}",
		},
		{
			name:             "OpenSearch with long index and document ID",
			path:             "/watchlist-entries-index/_doc/cobre_nit_012345648",
			method:           http.MethodPut,
			expectedSpanName: "PUT /watchlist-entries-index/_doc/{id}",
		},
		{
			name:             "Numeric ID",
			path:             "/api/orders/12345",
			method:           http.MethodGet,
			expectedSpanName: "GET /api/orders/{id}",
		},
		{
			name:             "Company ID with 2 letters",
			path:             "/events/ev_12345asdnkjb23",
			method:           http.MethodGet,
			expectedSpanName: "GET /events/{id}",
		},
		{
			name:             "Company ID with 3 letters",
			path:             "/clients/cli_2134asdn2n3",
			method:           http.MethodPost,
			expectedSpanName: "POST /clients/{id}",
		},
		{
			name:             "Company ID with 3 letters and 2 ids",
			path:             "/v1/accounts/acc_123asd/transactions/cli_2134asdn2n3",
			method:           http.MethodPost,
			expectedSpanName: "POST /v1/accounts/{id}/transactions/{id}",
		},
		{
			name:             "Company ID in middle of path",
			path:             "/users/usr_abc123def456ghi/profile",
			method:           http.MethodGet,
			expectedSpanName: "GET /users/{id}/profile",
		},
		{
			name:             "SHA-1 hash",
			path:             "/commits/356a192b7913b04c54574d18c28d46e6395428ab",
			method:           http.MethodGet,
			expectedSpanName: "GET /commits/{hash}",
		},
		{
			name:             "SHA-256 hash",
			path:             "/blobs/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			method:           http.MethodGet,
			expectedSpanName: "GET /blobs/{hash}",
		},
		{
			name:             "Base64 token",
			path:             "/sessions/aGVsbG93b3JsZGhlbGxvd29ybGRoZWxsb3dvcmxk",
			method:           http.MethodGet,
			expectedSpanName: "GET /sessions/{token}",
		},
		{
			name:             "ISO date",
			path:             "/logs/2024-01-15",
			method:           http.MethodGet,
			expectedSpanName: "GET /logs/{date}",
		},
		{
			name:             "Unix timestamp",
			path:             "/events/1705334400",
			method:           http.MethodGet,
			expectedSpanName: "GET /events/{timestamp}",
		},
		{
			name:             "Multiple dynamic segments",
			path:             "/users/123/orders/456",
			method:           http.MethodGet,
			expectedSpanName: "GET /users/{id}/orders/{id}",
		},
		{
			name:             "Static path unchanged",
			path:             "/api/health",
			method:           http.MethodGet,
			expectedSpanName: "GET /api/health",
		},
		{
			name:             "Path with <=10 segments works",
			path:             "/a/b/c/d/e/f/g/h/i/j",
			method:           http.MethodPost,
			expectedSpanName: "POST /a/b/c/d/e/f/g/h/i/j",
		},
		{
			name:             "Path with >10 segments falls back to method only",
			path:             "/a/b/c/d/e/f/g/h/i/j/k/l/m",
			method:           http.MethodPost,
			expectedSpanName: "POST",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spanName := SanitizeSpanName(tc.method, tc.path, nil)
			assert.Equal(t, tc.expectedSpanName, spanName)
		})
	}
}

func TestSanitizeSpanName_PerClientPatterns(t *testing.T) {
	defer setupMetricProvider(t)()

	config := &SpanNameConfig{
		AdditionalPatterns: []PathPattern{
			{
				Matcher:     regexp.MustCompile(`^/special/[^/]+/items/[^/]+$`),
				Replacement: "/special/{id}/items/{id}",
			},
		},
	}

	testCases := []struct {
		name             string
		path             string
		method           string
		expectedSpanName string
	}{
		{
			name:             "Per-client pattern matches",
			path:             "/special/abc/items/xyz",
			method:           http.MethodGet,
			expectedSpanName: "GET /special/{id}/items/{id}",
		},
		{
			name:             "Built-in heuristics still work",
			path:             "/users/123",
			method:           http.MethodGet,
			expectedSpanName: "GET /users/{id}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spanName := SanitizeSpanName(tc.method, tc.path, config)
			assert.Equal(t, tc.expectedSpanName, spanName)
		})
	}
}

func TestSanitizeSpanName_CustomSanitizer(t *testing.T) {
	defer setupMetricProvider(t)()

	config := &SpanNameConfig{
		CustomSanitizer: func(method, path string) string {
			return method // Use method-only
		},
	}

	testCases := []struct {
		name             string
		path             string
		method           string
		expectedSpanName string
	}{
		{
			name:             "Custom sanitizer overrides everything",
			path:             "/users/550e8400-e29b-41d4-a716-446655440000",
			method:           http.MethodGet,
			expectedSpanName: "GET",
		},
		{
			name:             "Custom sanitizer for POST",
			path:             "/api/orders/123",
			method:           http.MethodPost,
			expectedSpanName: "POST",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			spanName := SanitizeSpanName(tc.method, tc.path, config)
			assert.Equal(t, tc.expectedSpanName, spanName)
		})
	}
}

// Benchmark tests to measure sanitization performance

func BenchmarkSanitizeSpanName_BuiltInHeuristics(b *testing.B) {
	benchmarks := []struct {
		name   string
		method string
		path   string
	}{
		{"UUID", "GET", "/users/550e8400-e29b-41d4-a716-446655440000"},
		{"NumericID", "GET", "/orders/12345"},
		{"CompanyID", "GET", "/events/ev_12345asdnkjb23"},
		{"OpenSearchDoc", "PUT", "/my-index/_doc/doc_12345"},
		{"Hash", "GET", "/commits/356a192b7913b04c54574d18c28d46e6395428ab"},
		{"Token", "GET", "/sessions/aGVsbG93b3JsZGhlbGxvd29ybGRoZWxsb3dvcmxk"},
		{"StaticPath", "GET", "/api/health"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = SanitizeSpanName(bm.method, bm.path, nil)
			}
		})
	}
}

func BenchmarkSanitizeSpanName_WithPerClientPatterns(b *testing.B) {
	config := &SpanNameConfig{
		AdditionalPatterns: []PathPattern{
			{
				Matcher:     regexp.MustCompile(`^/special/[^/]+/items/[^/]+$`),
				Replacement: "/special/{id}/items/{id}",
			},
			{
				Matcher:     regexp.MustCompile(`^/custom/[^/]+/action$`),
				Replacement: "/custom/{id}/action",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeSpanName("GET", "/special/abc/items/xyz", config)
	}
}

func BenchmarkSanitizeSpanName_CustomSanitizer(b *testing.B) {
	config := &SpanNameConfig{
		CustomSanitizer: func(method, path string) string {
			return method
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeSpanName("GET", "/users/550e8400-e29b-41d4-a716-446655440000", config)
	}
}

// setupMetricProvider is a helper to set up the tracer and meter providers for tests.
// Note: This does NOT call metric.Init() because TestMain already initialized
// the metric package via sync.Once, and subsequent calls would be no-ops.
func setupMetricProvider(t testing.TB) func() {
	ctx := context.Background()
	res, err := resource.New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Set up tracer provider
	tracerProvider := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider for recording metrics in tests
	metricReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	return func() {
		_ = tracerProvider.Shutdown(context.Background())
		_ = meterProvider.Shutdown(context.Background())
		// Don't reset to nil - leave the TestMain providers in place
	}
}
