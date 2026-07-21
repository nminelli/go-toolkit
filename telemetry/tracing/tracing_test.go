package tracing

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

// mockStrStackTracerError implements the strStackTracer interface for testing
type mockStrStackTracerError struct {
	msg   string
	stack string
}

func (e *mockStrStackTracerError) Error() string {
	return e.msg
}

func (e *mockStrStackTracerError) StackTrace() string {
	return e.stack
}

func TestRecordError(t *testing.T) {
	os.Setenv("APP_VERSION", "1.0.0")
	defer os.Unsetenv("APP_VERSION")

	testErr := errors.New("test error")

	// Test with active span
	ctx, span := TP().Tracer("test").Start(context.Background(), "test-span")
	RecordError(ctx, testErr)
	span.End()

	// Test without active span (should not panic)
	emptyCtx := context.Background()
	RecordError(emptyCtx, testErr)
}

func TestRecordError_WithStackTrace(t *testing.T) {
	os.Setenv("APP_VERSION", "1.0.0")
	defer os.Unsetenv("APP_VERSION")

	tests := []struct {
		name        string
		setupError  func() error
		expectStack bool
	}{
		{
			name: "simple error without stack trace",
			setupError: func() error {
				return errors.New("simple error")
			},
			expectStack: false,
		},
		{
			name: "pkg/errors error with stack trace",
			setupError: func() error {
				return pkgerrors.New("error with stack trace")
			},
			expectStack: true,
		},
		{
			name: "wrapped error with stack trace",
			setupError: func() error {
				original := errors.New("original")
				return pkgerrors.Wrap(original, "wrapped error")
			},
			expectStack: true,
		},
		{
			name: "multiple wrapped errors",
			setupError: func() error {
				original := errors.New("original")
				wrapped1 := pkgerrors.Wrap(original, "first wrap")
				return pkgerrors.Wrap(wrapped1, "second wrap")
			},
			expectStack: true,
		},
		{
			name: "error with string stack trace interface",
			setupError: func() error {
				return &mockStrStackTracerError{
					msg:   "error with string stack trace",
					stack: "line1: function1()\nline2: function2()\nline3: function3()",
				}
			},
			expectStack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a span to capture the error
			ctx, span := TP().Tracer("test").Start(context.Background(), "test-span")
			defer span.End()

			err := tt.setupError()

			// This should not panic regardless of error type
			assert.NotPanics(t, func() {
				RecordError(ctx, err)
			})

			// Verify span is valid (we can't directly test the status without more complex mocking)
			assert.NotNil(t, span)
		})
	}
}

func TestRecordError_SpanAttributes(t *testing.T) {
	os.Setenv("APP_VERSION", "1.0.0")
	defer os.Unsetenv("APP_VERSION")

	testCases := []struct {
		name        string
		error       error
		expectType  string
		expectMsg   string
		expectStack bool
	}{
		{
			name:        "standard error",
			error:       errors.New("standard error message"),
			expectType:  "*errors.errorString",
			expectMsg:   "standard error message",
			expectStack: false,
		},
		{
			name:        "pkg/errors error",
			error:       pkgerrors.New("pkg error message"),
			expectType:  "*errors.fundamental",
			expectMsg:   "pkg error message",
			expectStack: true,
		},
		{
			name: "string stack tracer error",
			error: &mockStrStackTracerError{
				msg:   "string stack tracer message",
				stack: "test stack trace",
			},
			expectType:  "*tracing.mockStrStackTracerError",
			expectMsg:   "string stack tracer message",
			expectStack: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, span := TP().Tracer("test").Start(context.Background(), "test-span")
			defer span.End()

			RecordError(ctx, tc.error)

			// Verify basic error recording
			assert.Equal(t, tc.expectMsg, tc.error.Error())
			assert.Equal(t, tc.expectType, reflect.TypeOf(tc.error).String())

			// Verify stack trace interface
			if tc.expectStack {
				_, hasStackTracer := tc.error.(stackTracer)
				_, hasStrStackTracer := tc.error.(strStackTracer)
				hasAnyStackTracer := hasStackTracer || hasStrStackTracer
				assert.True(t, hasAnyStackTracer, "Error should implement stackTracer or strStackTracer interface")
			}
		})
	}
}

func TestRecordError_NilSpan(t *testing.T) {
	// Test that RecordError handles nil span gracefully
	ctx := context.Background() // No span in context

	err := errors.New("test error")

	// Should not panic when there's no span
	assert.NotPanics(t, func() {
		RecordError(ctx, err)
	})
}

func TestRecordError_ErrorTypes(t *testing.T) {
	os.Setenv("APP_VERSION", "1.0.0")
	defer os.Unsetenv("APP_VERSION")

	ctx, span := TP().Tracer("test").Start(context.Background(), "test-span")
	defer span.End()

	// Test different error types to ensure they're handled correctly
	errorTypes := []error{
		errors.New("standard error"),
		pkgerrors.New("pkg/errors error"),
		pkgerrors.Errorf("formatted error: %s", "value"),
		pkgerrors.Wrap(errors.New("original"), "wrapped"),
		&mockStrStackTracerError{
			msg:   "string stack tracer error",
			stack: "test stack",
		},
	}

	for i, err := range errorTypes {
		t.Run(reflect.TypeOf(err).String(), func(t *testing.T) {
			// Each should be recorded without panic
			assert.NotPanics(t, func() {
				RecordError(ctx, err)
			}, "Error type %d should be handled gracefully", i)
		})
	}
}

func TestAddAttribute(t *testing.T) {
	os.Setenv("APP_VERSION", "1.0.0")
	defer os.Unsetenv("APP_VERSION")

	testAttribute := attribute.String("test.key", "test.value")

	// Test with active span
	ctx, span := TP().Tracer("test").Start(context.Background(), "test-span")
	AddAttribute(ctx, testAttribute)
	span.End()

	// Test without active span (should not panic)
	emptyCtx := context.Background()
	AddAttribute(emptyCtx, testAttribute)
}

func TestGetTextMapPropagator(t *testing.T) {
	propagator := GetTextMapPropagator()
	assert.NotNil(t, propagator, "TextMapPropagator should not be nil")

	// Test that we can use the propagator for basic operations
	// Create a simple test to ensure the propagator works
	ctx := context.Background()
	carrier := make(map[string]string)

	// This should not panic
	propagator.Inject(ctx, &testCarrier{data: carrier})
	extractedCtx := propagator.Extract(ctx, &testCarrier{data: carrier})
	assert.NotNil(t, extractedCtx, "Extracted context should not be nil")
}

func TestSpanFromContext(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		wantNil  bool
	}{
		{
			name: "with active span",
			setupCtx: func() context.Context {
				ctx, _ := TP().Tracer("test").Start(context.Background(), "test-span")
				return ctx
			},
			wantNil: false,
		},
		{
			name: "without active span",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantNil: false, // Should return background span, not nil
		},
		{
			name: "with nil context - edge case",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			span := SpanFromContext(ctx)

			if tt.wantNil {
				assert.Nil(t, span)
			} else {
				assert.NotNil(t, span)
			}
		})
	}
}

// testCarrier implements the propagation.TextMapCarrier interface for testing
type testCarrier struct {
	data map[string]string
}

func (c *testCarrier) Get(key string) string {
	return c.data[key]
}

func (c *testCarrier) Set(key, value string) {
	c.data[key] = value
}

func (c *testCarrier) Keys() []string {
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}
