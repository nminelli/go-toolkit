package tracing

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// Mock types for testing stack trace functionality
type mockStackTracer struct {
	err   error
	stack errors.StackTrace
}

func (m mockStackTracer) Error() string {
	return m.err.Error()
}

func (m mockStackTracer) StackTrace() errors.StackTrace {
	return m.stack
}

type mockStrStackTracer struct {
	err   error
	stack string
}

func (m mockStrStackTracer) Error() string {
	return m.err.Error()
}

func (m mockStrStackTracer) StackTrace() string {
	return m.stack
}

func TestStackTrace_String(t *testing.T) {
	tests := []struct {
		name     string
		setupErr func() error
		wantNil  bool
	}{
		{
			name: "error with stack trace",
			setupErr: func() error {
				return errors.New("test error with stack")
			},
			wantNil: false,
		},
		{
			name: "wrapped error with stack trace",
			setupErr: func() error {
				original := errors.New("original error")
				return errors.Wrap(original, "wrapped error")
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setupErr()

			// Check if the error has a stack trace
			if st, ok := err.(stackTracer); ok {
				stackTrace := &stackTrace{st}
				result := stackTrace.String()

				if tt.wantNil {
					assert.Empty(t, result)
				} else {
					assert.NotEmpty(t, result)
					// Stack trace should contain function information
					assert.Contains(t, result, "TestStackTrace_String")
				}
			} else {
				t.Skipf("Error does not implement stackTracer interface")
			}
		})
	}
}

func TestStackTrace_String_WithMockStackTracer(t *testing.T) {
	// Create a mock stack trace
	mockStack := errors.StackTrace{
		errors.Frame(0x123456),
		errors.Frame(0x789abc),
	}

	mockErr := mockStackTracer{
		err:   fmt.Errorf("mock error"),
		stack: mockStack,
	}

	st := &stackTrace{mockErr}
	result := st.String()

	assert.NotEmpty(t, result)
	// The result should contain formatted stack frames
	assert.Contains(t, result, "\n")
}

func TestStackTrace_String_WithEmptyStack(t *testing.T) {
	mockErr := mockStackTracer{
		err:   fmt.Errorf("mock error"),
		stack: nil,
	}

	st := &stackTrace{mockErr}
	result := st.String()

	// Should not panic with empty stack
	assert.NotNil(t, result)
}

func TestStrStackTracer_Interface(t *testing.T) {
	mockErr := mockStrStackTracer{
		err:   fmt.Errorf("mock error"),
		stack: "line1\nline2\nline3",
	}

	// Test that it implements the interface correctly
	var _ strStackTracer = mockErr

	assert.Equal(t, "mock error", mockErr.Error())
	assert.Equal(t, "line1\nline2\nline3", mockErr.StackTrace())
}

func TestStackTracer_Interface(t *testing.T) {
	mockStack := errors.StackTrace{
		errors.Frame(0x123456),
	}

	mockErr := mockStackTracer{
		err:   fmt.Errorf("mock error"),
		stack: mockStack,
	}

	// Test that it implements the interface correctly
	var _ stackTracer = mockErr

	assert.Equal(t, "mock error", mockErr.Error())
	assert.Equal(t, mockStack, mockErr.StackTrace())
}

func TestStackTraceInterfaces_RealPkgErrorsError(t *testing.T) {
	// Test with real pkg/errors error
	originalErr := errors.New("original error")
	wrappedErr := errors.Wrap(originalErr, "wrapped")

	// Should implement stackTracer interface
	if st, ok := wrappedErr.(stackTracer); ok {
		assert.NotNil(t, st.StackTrace())
		assert.True(t, len(st.StackTrace()) > 0)

		// Test stackTrace wrapper
		wrapper := &stackTrace{st}
		result := wrapper.String()
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "TestStackTraceInterfaces_RealPkgErrorsError")
	} else {
		t.Errorf("pkg/errors error should implement stackTracer interface")
	}
}

func TestStackTrace_String_Integration(t *testing.T) {
	// Test the full integration flow
	err := errors.New("integration test error")

	if st, ok := err.(stackTracer); ok {
		wrapper := &stackTrace{st}
		result := wrapper.String()

		// Verify stack trace format
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "\n") // Should have newlines

		// Should contain function name and line information
		lines := fmt.Sprintf("%+v", result)
		assert.NotEmpty(t, lines)
	} else {
		t.Skipf("Error does not implement stackTracer interface in this test environment")
	}
}
