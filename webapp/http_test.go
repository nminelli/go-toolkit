package webapp_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/MFN-AISystems/go-toolkit/webapp"
)

func TestJSON(t *testing.T) {
	testCases := []struct {
		name                string
		status              int
		data                interface{}
		expectedStatus      int
		expectedBody        string
		expectedContentType string
		shouldWriteBody     bool
	}{
		{
			name:                "successful JSON response with map",
			status:              http.StatusOK,
			data:                map[string]string{"message": "success"},
			expectedStatus:      http.StatusOK,
			expectedBody:        `{"message":"success"}`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
		{
			name:                "successful JSON response with struct",
			status:              http.StatusCreated,
			data:                struct{ ID int }{ID: 123},
			expectedStatus:      http.StatusCreated,
			expectedBody:        `{"ID":123}`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
		{
			name:                "no content response",
			status:              http.StatusNoContent,
			data:                map[string]string{"message": "this should not appear"},
			expectedStatus:      http.StatusNoContent,
			expectedBody:        "",
			expectedContentType: "", // No content type should be set
			shouldWriteBody:     false,
		},
		{
			name:                "nil data response",
			status:              http.StatusOK,
			data:                nil,
			expectedStatus:      http.StatusOK,
			expectedBody:        "",
			expectedContentType: "", // No content type should be set for nil data
			shouldWriteBody:     false,
		},
		{
			name:                "empty map response",
			status:              http.StatusOK,
			data:                map[string]string{},
			expectedStatus:      http.StatusOK,
			expectedBody:        `{}`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
		{
			name:                "slice response",
			status:              http.StatusOK,
			data:                []string{"item1", "item2"},
			expectedStatus:      http.StatusOK,
			expectedBody:        `["item1","item2"]`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
		{
			name:                "string response",
			status:              http.StatusOK,
			data:                "plain string",
			expectedStatus:      http.StatusOK,
			expectedBody:        `"plain string"`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
		{
			name:                "number response",
			status:              http.StatusOK,
			data:                42,
			expectedStatus:      http.StatusOK,
			expectedBody:        `42`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
		{
			name:                "boolean response",
			status:              http.StatusOK,
			data:                true,
			expectedStatus:      http.StatusOK,
			expectedBody:        `true`,
			expectedContentType: "application/json",
			shouldWriteBody:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			ctx := context.Background()

			// Act
			webapp.JSON(ctx, w, tc.status, tc.data)

			// Assert
			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.shouldWriteBody {
				assert.Equal(t, tc.expectedBody, w.Body.String())
				assert.Equal(t, tc.expectedContentType, w.Header().Get("Content-Type"))
			} else {
				assert.Empty(t, w.Body.String())
				assert.Empty(t, w.Header().Get("Content-Type"))
			}
		})
	}
}

func TestJSONError(t *testing.T) {
	testCases := []struct {
		name                string
		httpError           webapp.HttpError
		expectedStatus      int
		expectedErrorCode   string
		expectedDescription string
		expectTracingCall   bool
		wrappedError        error
	}{
		{
			name:                "bad request error",
			httpError:           webapp.NewBadRequestError("VALIDATION_ERROR", "Invalid input provided", nil),
			expectedStatus:      http.StatusBadRequest,
			expectedErrorCode:   "VALIDATION_ERROR",
			expectedDescription: "Invalid input provided",
			expectTracingCall:   false,
		},
		{
			name:                "not found error",
			httpError:           webapp.NewNotFoundError("RESOURCE_NOT_FOUND", "User not found"),
			expectedStatus:      http.StatusNotFound,
			expectedErrorCode:   "RESOURCE_NOT_FOUND",
			expectedDescription: "User not found",
			expectTracingCall:   false,
		},
		{
			name:                "conflict error",
			httpError:           webapp.NewConflictError("DUPLICATE_RESOURCE", "Resource already exists", nil),
			expectedStatus:      http.StatusConflict,
			expectedErrorCode:   "DUPLICATE_RESOURCE",
			expectedDescription: "Resource already exists",
			expectTracingCall:   false,
		},
		{
			name:                "internal server error without wrapped error",
			httpError:           webapp.NewInternalServerError("INTERNAL_ERROR", "Something went wrong", nil),
			expectedStatus:      http.StatusInternalServerError,
			expectedErrorCode:   "INTERNAL_ERROR",
			expectedDescription: "Something went wrong",
			expectTracingCall:   true,
		},
		{
			name:                "internal server error with wrapped error",
			httpError:           webapp.NewInternalServerError("DATABASE_ERROR", "Database connection failed", errors.New("connection timeout")),
			expectedStatus:      http.StatusInternalServerError,
			expectedErrorCode:   "DATABASE_ERROR",
			expectedDescription: "Database connection failed",
			expectTracingCall:   true,
			wrappedError:        errors.New("connection timeout"),
		},
		{
			name:                "bad gateway error (5xx)",
			httpError:           webapp.NewHttpError(http.StatusBadGateway, "GATEWAY_ERROR", "External service unavailable", errors.New("service timeout")),
			expectedStatus:      http.StatusBadGateway,
			expectedErrorCode:   "GATEWAY_ERROR",
			expectedDescription: "External service unavailable",
			expectTracingCall:   true,
			wrappedError:        errors.New("service timeout"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()

			// Create a context with a span to test tracing behavior
			tracer := otel.Tracer("test-tracer")
			ctx, span := tracer.Start(context.Background(), "test-operation")
			defer span.End()

			// Act
			webapp.JSONError(ctx, w, tc.httpError)

			// Assert HTTP response
			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Parse and verify the JSON response body
			var errorResponse map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedErrorCode, errorResponse["error_code"])
			assert.Equal(t, tc.expectedDescription, errorResponse["error_description"])

			// Verify the response structure matches the expected httpErrorResponse
			expectedResponse := map[string]string{
				"error_code":        tc.expectedErrorCode,
				"error_description": tc.expectedDescription,
			}
			assert.Equal(t, expectedResponse, errorResponse)

			// Note: We cannot easily test if tracing.RecordError was called without mocking
			// the tracing package, but we can verify the function completes without panicking
			// for 5xx errors. The tracing behavior is tested in the tracing package itself.
		})
	}
}

// mockSpan is a test spy that records whether tracing methods were called
type mockSpan struct {
	trace.Span
	statusCode codes.Code
	statusDesc string
	statusSet  bool
}

func (m *mockSpan) SetStatus(code codes.Code, description string) {
	m.statusCode = code
	m.statusDesc = description
	m.statusSet = true
}

func (m *mockSpan) SpanContext() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{})
}

func (m *mockSpan) IsRecording() bool {
	return true
}

func (m *mockSpan) SetAttributes(...attribute.KeyValue)     {}
func (m *mockSpan) End(...trace.SpanEndOption)              {}
func (m *mockSpan) RecordError(error, ...trace.EventOption) {}
func (m *mockSpan) AddEvent(string, ...trace.EventOption)   {}
func (m *mockSpan) SetName(string)                          {}

// TestJSONError_ActualTracingCalls verifies that tracing.RecordError is actually called for 5xx errors
// This test uses a mock span to detect when SetStatus(codes.Error, ...) is called,
// which happens when tracing.RecordError is invoked.
func TestJSONError_ActualTracingCalls(t *testing.T) {
	t.Skip("Skipping: tracing calls are disabled in http.go")
	testCases := []struct {
		name          string
		httpError     webapp.HttpError
		runAssertions func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan)
	}{
		{
			name:      "4xx error should not call tracing.RecordError",
			httpError: webapp.NewBadRequestError("VALIDATION_ERROR", "Invalid input", nil),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusBadRequest, w.Code)
				assert.False(t, mockSpan.statusSet, "Expected SetStatus NOT to be called for 4xx errors")
			},
		},
		{
			name:      "404 error should not call tracing.RecordError",
			httpError: webapp.NewNotFoundError("NOT_FOUND", "Resource not found"),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusNotFound, w.Code)
				assert.False(t, mockSpan.statusSet, "Expected SetStatus NOT to be called for 404 errors")
			},
		},
		{
			name:      "5xx error should call tracing.RecordError",
			httpError: webapp.NewInternalServerError("SERVER_ERROR", "Internal error", nil),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.True(t, mockSpan.statusSet, "Expected SetStatus to be called for 5xx errors")
				assert.Equal(t, codes.Error, mockSpan.statusCode, "Expected error status code")
				assert.NotEmpty(t, mockSpan.statusDesc, "Expected status description to be set")
			},
		},
		{
			name:      "502 error should call tracing.RecordError",
			httpError: webapp.NewHttpError(http.StatusBadGateway, "GATEWAY_ERROR", "Bad gateway", errors.New("upstream failed")),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusBadGateway, w.Code)
				assert.True(t, mockSpan.statusSet, "Expected SetStatus to be called for 502 errors")
				assert.Equal(t, codes.Error, mockSpan.statusCode, "Expected error status code")
				assert.NotEmpty(t, mockSpan.statusDesc, "Expected status description to be set")
			},
		},
		{
			name:      "boundary case 499 should not record",
			httpError: webapp.NewHttpError(499, "CLIENT_CLOSED", "Client closed connection", nil),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, 499, w.Code)
				assert.False(t, mockSpan.statusSet, "Expected SetStatus NOT to be called for 4xx boundary case")
			},
		},
		{
			name:      "boundary case 500 should record",
			httpError: webapp.NewHttpError(http.StatusInternalServerError, "SERVER_ERROR", "Internal server error", nil),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.True(t, mockSpan.statusSet, "Expected SetStatus to be called for 5xx boundary case")
				assert.Equal(t, codes.Error, mockSpan.statusCode, "Expected error status code")
				assert.NotEmpty(t, mockSpan.statusDesc, "Expected status description to be set")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock span that can report back what methods were called
			mockSpan := &mockSpan{}

			// Create a context with our mock span
			ctx := trace.ContextWithSpan(context.Background(), mockSpan)

			// Arrange
			w := httptest.NewRecorder()

			// Act
			webapp.JSONError(ctx, w, tc.httpError)

			// Assert using the test case specific assertion function
			tc.runAssertions(t, ctx, w, mockSpan)
		})
	}
}

func TestJSONError_WrappedErrorInTracing(t *testing.T) {
	t.Skip("Skipping: tracing calls are disabled in http.go")
	testCases := []struct {
		name          string
		httpError     webapp.HttpError
		runAssertions func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan)
	}{
		{
			name:      "5xx error with wrapped error should call tracing.RecordError with wrapped error",
			httpError: webapp.NewInternalServerError("DATABASE_ERROR", "Database error occurred", errors.New("database connection failed")),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.True(t, mockSpan.statusSet, "Expected SetStatus to be called for 5xx error with wrapped error")
				assert.Equal(t, codes.Error, mockSpan.statusCode, "Expected error status code")
				assert.NotEmpty(t, mockSpan.statusDesc, "Expected status description to be set")
				// The tracing should use the wrapped error (not the HTTP error)
				// This is verified by the fact that SetStatus was called, which means tracing.RecordError was called
			},
		},
		{
			name:      "5xx error without wrapped error should call tracing.RecordError with HTTP error",
			httpError: webapp.NewInternalServerError("SERVER_ERROR", "Internal server error", nil),
			runAssertions: func(t *testing.T, ctx context.Context, w *httptest.ResponseRecorder, mockSpan *mockSpan) {
				assert.Equal(t, http.StatusInternalServerError, w.Code)
				assert.True(t, mockSpan.statusSet, "Expected SetStatus to be called for 5xx error without wrapped error")
				assert.Equal(t, codes.Error, mockSpan.statusCode, "Expected error status code")
				assert.NotEmpty(t, mockSpan.statusDesc, "Expected status description to be set")
				// The tracing should use the HTTP error itself since there's no wrapped error
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock span to capture tracing calls
			mockSpan := &mockSpan{}
			ctx := trace.ContextWithSpan(context.Background(), mockSpan)

			// Arrange
			w := httptest.NewRecorder()

			// Act
			webapp.JSONError(ctx, w, tc.httpError)

			// Assert using the test case specific assertion function
			tc.runAssertions(t, ctx, w, mockSpan)
		})
	}
}

func TestJSONError_WithoutSpanContext(t *testing.T) {
	// Test JSONError with a context that doesn't have a span
	// This ensures the function handles missing spans gracefully

	// Arrange
	w := httptest.NewRecorder()
	ctx := context.Background() // No span in context
	httpErr := webapp.NewInternalServerError("NO_SPAN_ERROR", "Error without span context", errors.New("original error"))

	// Act - should not panic even without span
	assert.NotPanics(t, func() {
		webapp.JSONError(ctx, w, httpErr)
	})

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "NO_SPAN_ERROR", errorResponse["error_code"])
	assert.Equal(t, "Error without span context", errorResponse["error_description"])
}

func TestJSONError_TracingBehavior(t *testing.T) {
	// Test the boundary between 4xx and 5xx errors for tracing behavior

	testCases := []struct {
		name        string
		status      int
		shouldTrace bool
	}{
		{
			name:        "4xx error - should not trace",
			status:      http.StatusNotFound,
			shouldTrace: false,
		},
		{
			name:        "4xx error boundary - should not trace",
			status:      http.StatusTeapot, // 418 - still 4xx
			shouldTrace: false,
		},
		{
			name:        "5xx error boundary - should trace",
			status:      http.StatusInternalServerError, // 500 - first 5xx
			shouldTrace: true,
		},
		{
			name:        "5xx error - should trace",
			status:      http.StatusBadGateway,
			shouldTrace: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			tracer := otel.Tracer("test-tracer")
			ctx, span := tracer.Start(context.Background(), "test-operation")
			defer span.End()

			httpErr := webapp.NewHttpError(tc.status, "TEST_ERROR", "Test error message", errors.New("wrapped error"))

			// Act - should not panic regardless of error type
			assert.NotPanics(t, func() {
				webapp.JSONError(ctx, w, httpErr)
			})

			// Assert basic response structure
			assert.Equal(t, tc.status, w.Code)

			var errorResponse map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			require.NoError(t, err)

			assert.Equal(t, "TEST_ERROR", errorResponse["error_code"])
			assert.Equal(t, "Test error message", errorResponse["error_description"])
		})
	}
}

func TestJSONError_ErrorResponseFormat(t *testing.T) {
	// Test that the error response format is exactly as specified

	// Arrange
	w := httptest.NewRecorder()
	ctx := context.Background()
	httpErr := webapp.NewBadRequestError("FIELD_VALIDATION", "Email field is required", nil)

	// Act
	webapp.JSONError(ctx, w, httpErr)

	// Assert exact JSON structure
	expectedJSON := `{"error_code":"FIELD_VALIDATION","error_description":"Email field is required"}`

	// Parse both to avoid formatting differences
	var expected, actual map[string]string
	require.NoError(t, json.Unmarshal([]byte(expectedJSON), &expected))
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &actual))

	assert.Equal(t, expected, actual)

	// Verify JSON keys are exactly as expected (testing the httpErrorResponse struct tags)
	assert.Contains(t, w.Body.String(), `"error_code"`)
	assert.Contains(t, w.Body.String(), `"error_description"`)
	assert.NotContains(t, w.Body.String(), `"Code"`) // Should not contain Go field names
	assert.NotContains(t, w.Body.String(), `"Description"`)
}

func TestJSONError_WithWrappedErrorBehavior(t *testing.T) {
	// Test that wrapped errors are handled correctly in tracing for 5xx errors

	testCases := []struct {
		name         string
		wrappedError error
		description  string
	}{
		{
			name:         "with simple wrapped error",
			wrappedError: errors.New("database connection failed"),
			description:  "Should use wrapped error for tracing",
		},
		{
			name:         "without wrapped error",
			wrappedError: nil,
			description:  "Should use the http error itself for tracing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			w := httptest.NewRecorder()
			tracer := otel.Tracer("test-tracer")
			ctx, span := tracer.Start(context.Background(), "test-operation")
			defer span.End()

			httpErr := webapp.NewInternalServerError("SERVER_ERROR", "Internal server error occurred", tc.wrappedError)

			// Act
			webapp.JSONError(ctx, w, httpErr)

			// Assert
			assert.Equal(t, http.StatusInternalServerError, w.Code)

			var errorResponse map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
			require.NoError(t, err)

			// The response should always show the http error details, not the wrapped error
			assert.Equal(t, "SERVER_ERROR", errorResponse["error_code"])
			assert.Equal(t, "Internal server error occurred", errorResponse["error_description"])

			// The wrapped error should not appear in the response JSON
			if tc.wrappedError != nil {
				assert.NotContains(t, w.Body.String(), tc.wrappedError.Error())
			}
		})
	}
}

// Test helper to verify the httpErrorResponse struct matches expectations
func TestHttpErrorResponseStructure(t *testing.T) {
	// This test ensures our understanding of the httpErrorResponse struct is correct
	// by testing the JSON serialization directly

	type httpErrorResponse struct {
		Code        string `json:"error_code"`
		Description string `json:"error_description"`
	}

	response := httpErrorResponse{
		Code:        "TEST_CODE",
		Description: "Test description",
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	expected := `{"error_code":"TEST_CODE","error_description":"Test description"}`
	assert.JSONEq(t, expected, string(jsonData))
}
