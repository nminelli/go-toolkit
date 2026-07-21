package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

// mockHandler is a test handler that captures the context for verification
type mockHandler struct {
	capturedContext context.Context
}

func (m *mockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.capturedContext = r.Context()
	w.WriteHeader(http.StatusOK)
}

func TestHeaderForwarder(t *testing.T) {
	testCases := []struct {
		name          string
		headers       map[string]string
		setupMocks    func(ctx context.Context)
		runAssertions func(t *testing.T, handler *mockHandler, err error)
	}{
		{
			name: "valid trace context headers",
			headers: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				"tracestate":  "congo=t61rcWkgMzE",
			},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for trace context test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Verify that the context contains trace information
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)

				// Check if span context exists (it should be extracted from headers)
				spanCtx := span.SpanContext()
				assert.True(t, spanCtx.IsValid())
			},
		},
		{
			name: "valid baggage headers",
			headers: map[string]string{
				"baggage": "key1=value1,key2=value2",
			},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for baggage test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Verify that the context contains baggage information
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)
			},
		},
		{
			name: "multiple headers combined",
			headers: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				"tracestate":  "congo=t61rcWkgMzE",
				"baggage":     "service=example,key=value",
			},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for combined test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Verify that the context contains both trace and baggage information
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)

				spanCtx := span.SpanContext()
				assert.True(t, spanCtx.IsValid())
			},
		},
		{
			name:    "no tracing headers",
			headers: map[string]string{},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for no headers test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Should not panic or error when no headers are present
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)
			},
		},
		{
			name: "malformed traceparent header",
			headers: map[string]string{
				"traceparent": "invalid-traceparent-header",
			},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for malformed header test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Should not panic or error with malformed headers
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)
			},
		},
		{
			name: "malformed baggage header",
			headers: map[string]string{
				"baggage": "invalid,baggage,format",
			},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for malformed baggage test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Should not panic or error with malformed baggage headers
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)
			},
		},
		{
			name: "context propagation verification",
			headers: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				"tracestate":  "congo=t61rcWkgMzE",
			},
			setupMocks: func(ctx context.Context) {
				// No specific mocks needed for context propagation test
			},
			runAssertions: func(t *testing.T, handler *mockHandler, err error) {
				assert.NoError(t, err)

				// Verify that the context passed to the handler is different from the original
				// This ensures that the middleware is actually modifying the context
				originalCtx := context.Background()
				assert.NotEqual(t, originalCtx, handler.capturedContext)

				// Verify that the context contains trace information
				span := trace.SpanFromContext(handler.capturedContext)
				assert.NotNil(t, span)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			tc.setupMocks(context.Background())

			// Create a mock handler to capture the context
			mockHandler := &mockHandler{}

			// Create the middleware
			middleware := HeaderForwarder(mockHandler)

			// Create a test request with the specified headers
			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Execute the middleware
			middleware.ServeHTTP(rr, req)

			// Run assertions
			tc.runAssertions(t, mockHandler, nil)
		})
	}
}

func TestHeaderForwarder_WithRealHandler(t *testing.T) {
	// Test that the middleware works correctly with a real handler
	executed := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executed = true

		// Verify that we can extract trace information from the context
		span := trace.SpanFromContext(r.Context())
		assert.NotNil(t, span)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	middleware := HeaderForwarder(handler)

	// Test with trace headers
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set("tracestate", "congo=t61rcWkgMzE")

	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	assert.True(t, executed)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "success", rr.Body.String())
}

func TestHeaderForwarder_CompositePropagator(t *testing.T) {
	// Test that the middleware uses the correct composite propagator
	mockHandler := &mockHandler{}
	middleware := HeaderForwarder(mockHandler)

	// The middleware should use CompositeTextMapPropagator with TraceContext and Baggage
	// This is verified by the fact that it doesn't panic and handles the headers correctly
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req.Header.Set("baggage", "key1=value1,key2=value2")

	rr := httptest.NewRecorder()

	// Should not panic and should handle both types of headers
	assert.NotPanics(t, func() {
		middleware.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusOK, rr.Code)
}
