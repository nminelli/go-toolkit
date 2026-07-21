package webapp_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MFN-AISystems/go-toolkit/webapp"
	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type strStackTracer interface {
	StackTrace() string
}

func TestFormatPanic_NilValue(t *testing.T) {
	panicInfo := webapp.FormatPanic(nil)
	assert.Nil(t, panicInfo, "Should return nil for nil panic value")
}

func TestFormatPanic_StringPanic(t *testing.T) {
	panicMessage := "test string panic"
	panicInfo := webapp.FormatPanic(panicMessage)

	require.NotNil(t, panicInfo, "Should format string panic")
	assert.Equal(t, panicMessage, panicInfo.Value)
	assert.Contains(t, panicInfo.Message, panicMessage)
	assert.Contains(t, panicInfo.Message, "panic:")
	assert.NotEmpty(t, panicInfo.StackTrace)
	assert.NotNil(t, panicInfo.Error)
	assert.Contains(t, panicInfo.Error.Error(), panicMessage)
}

func TestFormatPanic_ErrorPanic(t *testing.T) {
	panicErr := errors.New("test error panic")
	panicInfo := webapp.FormatPanic(panicErr)

	require.NotNil(t, panicInfo, "Should format error panic")
	assert.Equal(t, panicErr, panicInfo.Value)
	assert.Contains(t, panicInfo.Message, panicErr.Error())
	assert.Contains(t, panicInfo.Message, "panic:")
	assert.NotEmpty(t, panicInfo.StackTrace)
	assert.NotNil(t, panicInfo.Error)
	assert.Contains(t, panicInfo.Error.Error(), panicErr.Error())
}

func TestFormatPanic_CustomTypePanic(t *testing.T) {
	customPanic := struct {
		Code    int
		Message string
	}{
		Code:    500,
		Message: "custom error",
	}
	panicInfo := webapp.FormatPanic(customPanic)

	require.NotNil(t, panicInfo, "Should format custom type panic")
	assert.Equal(t, customPanic, panicInfo.Value)
	assert.Contains(t, panicInfo.Message, "panic:")
	assert.Contains(t, panicInfo.Message, "500")
	assert.Contains(t, panicInfo.Message, "custom error")
	assert.NotEmpty(t, panicInfo.StackTrace)
	assert.NotNil(t, panicInfo.Error)
}

func TestFormatPanic_IntegerPanic(t *testing.T) {
	panicValue := 42
	panicInfo := webapp.FormatPanic(panicValue)

	require.NotNil(t, panicInfo, "Should format integer panic")
	assert.Equal(t, panicValue, panicInfo.Value)
	assert.Contains(t, panicInfo.Message, "42")
	assert.Contains(t, panicInfo.Message, "panic:")
	assert.NotEmpty(t, panicInfo.StackTrace)
	assert.NotNil(t, panicInfo.Error)
}

func TestPanicInfo_Structure(t *testing.T) {
	panicMessage := "structured test"
	panicInfo := webapp.FormatPanic(panicMessage)

	require.NotNil(t, panicInfo)

	// Verify all fields are populated
	assert.NotNil(t, panicInfo.Value)
	assert.NotEmpty(t, panicInfo.Message)
	assert.NotEmpty(t, panicInfo.StackTrace)
	assert.NotNil(t, panicInfo.Error)

	// Verify message format
	assert.True(t, strings.HasPrefix(panicInfo.Message, "panic:"))

	// Verify error wrapping
	assert.Contains(t, panicInfo.Error.Error(), "panic:")
	assert.Contains(t, panicInfo.Error.Error(), panicMessage)
}

func TestFormatPanic_SafetyNeverPanics(t *testing.T) {
	// Test with various edge cases to ensure the function never panics
	testCases := []struct {
		name       string
		panicValue interface{}
		expectNil  bool
	}{
		{"empty string", "", false},
		{"nil value", nil, true},
		{"zero int", 0, false},
		{"complex struct", map[string]interface{}{"nested": map[string]string{"key": "value"}}, false},
		{"function", func() {}, false},
		{"channel", make(chan int), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This test should never panic, regardless of input
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("FormatPanic panicked with input %v: %v", tc.panicValue, r)
				}
			}()

			panicInfo := webapp.FormatPanic(tc.panicValue)

			if tc.expectNil {
				assert.Nil(t, panicInfo)
			} else {
				assert.NotNil(t, panicInfo)
				// For function types, we can't use Equal comparison
				if tc.name == "function" {
					assert.NotNil(t, panicInfo.Value)
				} else {
					assert.Equal(t, tc.panicValue, panicInfo.Value)
				}
			}
		})
	}
}

func TestTraditionalUsagePattern(t *testing.T) {
	// Test the traditional usage pattern similar to existing telemetry middleware
	panicMessage := "traditional usage test"
	var panicInfo *webapp.PanicInfo

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicInfo = webapp.FormatPanic(r)
			}
		}()
		panic(panicMessage)
	}()

	require.NotNil(t, panicInfo)
	assert.Equal(t, panicMessage, panicInfo.Value)
	assert.Contains(t, panicInfo.Message, panicMessage)
	assert.NotEmpty(t, panicInfo.StackTrace)
	assert.NotNil(t, panicInfo.Error)
}

// Tests for embedded stack trace functionality
func TestPanicInfo_ErrorWithEmbeddedStackTrace(t *testing.T) {
	testCases := []struct {
		name       string
		panicValue interface{}
	}{
		{
			name:       "string panic",
			panicValue: "test panic",
		},
		{
			name:       "error panic",
			panicValue: errors.New("test error"),
		},
		{
			name:       "wrapped error panic",
			panicValue: pkgerrors.Wrap(errors.New("root cause"), "wrapper"),
		},
		{
			name:       "integer panic",
			panicValue: 42,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			panicInfo := webapp.FormatPanic(tc.panicValue)
			require.NotNil(t, panicInfo)

			// Error should exist
			assert.NotNil(t, panicInfo.Error)

			// Should be able to extract stack trace from the error
			stackTracer, ok := panicInfo.Error.(strStackTracer)
			assert.True(t, ok, "Error should have embedded stack trace")

			// Stack trace from error should match the one in PanicInfo
			assert.Equal(t, panicInfo.StackTrace, stackTracer.StackTrace(),
				"Stack trace from error should match PanicInfo.StackTrace")
		})
	}
}

func TestPanicError_ErrorInterface(t *testing.T) {
	panicInfo := webapp.FormatPanic("test panic")
	err := panicInfo.Error

	// Should implement error interface
	assert.NotEmpty(t, err.Error())

	// Should implement Unwrap for error chaining
	if unwrappable, ok := err.(interface{ Unwrap() error }); ok {
		cause := unwrappable.Unwrap()
		assert.NotNil(t, cause)
	}

	// Should implement Cause for pkg/errors compatibility
	if causer, ok := err.(interface{ Cause() error }); ok {
		cause := causer.Cause()
		assert.NotNil(t, cause)
	}
}

func TestErrorWrapping_PreservesOriginalError(t *testing.T) {
	originalErr := errors.New("original error message")
	panicInfo := webapp.FormatPanic(originalErr)

	// Should be able to unwrap to get original error
	if unwrappable, ok := panicInfo.Error.(interface{ Unwrap() error }); ok {
		unwrapped := unwrappable.Unwrap()
		assert.Equal(t, originalErr.Error(), unwrapped.Error())
	}

	// Should work with errors.Unwrap
	unwrapped := errors.Unwrap(panicInfo.Error)
	if unwrapped != nil {
		assert.Equal(t, originalErr.Error(), unwrapped.Error())
	}
}

func TestPracticalUsageExample(t *testing.T) {
	// Simulate a panic and recovery
	panicInfo := webapp.FormatPanic(errors.New("database connection failed"))

	// Now you can pass the error around and still access the stack trace
	err := panicInfo.Error

	// Later, when you need the stack trace:
	stackTracer, ok := panicInfo.Error.(strStackTracer)
	assert.True(t, ok, "Error should have embedded stack trace")
	stackTrace := stackTracer.StackTrace()

	// The stack trace should not be empty and should contain some meaningful information
	assert.NotEmpty(t, stackTrace, "Stack trace should not be empty")

	// The error message should be descriptive
	assert.Contains(t, err.Error(), "panic:")
	assert.Contains(t, err.Error(), "database connection failed")

	t.Logf("Error message: %s", err.Error())
	t.Logf("Embedded stack trace:\n%s", stackTrace)
}

// Benchmark tests to ensure panic recovery is not too expensive
func BenchmarkFormatPanic(b *testing.B) {
	panicValue := "benchmark panic"
	for i := 0; i < b.N; i++ {
		webapp.FormatPanic(panicValue)
	}
}
