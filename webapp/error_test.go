package webapp_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MFN-AISystems/go-toolkit/webapp"
)

func TestHttpError_Error(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		code        string
		description string
		wrapped     error
		expected    string
	}{
		{
			name:        "error without wrapped error",
			status:      http.StatusBadRequest,
			code:        "INVALID_REQUEST",
			description: "The request is invalid",
			wrapped:     nil,
			expected:    "INVALID_REQUEST: The request is invalid",
		},
		{
			name:        "error with wrapped error",
			status:      http.StatusInternalServerError,
			code:        "DATABASE_ERROR",
			description: "Failed to connect to database",
			wrapped:     errors.New("connection timeout"),
			expected:    "DATABASE_ERROR: Failed to connect to database: connection timeout",
		},
		{
			name:        "error with empty description",
			status:      http.StatusNotFound,
			code:        "NOT_FOUND",
			description: "",
			wrapped:     nil,
			expected:    "NOT_FOUND: ",
		},
		{
			name:        "error with empty code",
			status:      http.StatusConflict,
			code:        "",
			description: "Resource already exists",
			wrapped:     nil,
			expected:    ": Resource already exists",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			err := webapp.NewHttpError(tc.status, tc.code, tc.description, tc.wrapped)

			// Act
			result := err.Error()

			// Assert
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHttpError_HttpStatus(t *testing.T) {
	testCases := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "bad request",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "internal server error",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "conflict",
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			err := webapp.NewHttpError(tc.expectedStatus, "TEST_CODE", "Test description", nil)

			// Act
			status := err.HttpStatus()

			// Assert
			assert.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestHttpError_Code(t *testing.T) {
	testCases := []struct {
		name         string
		expectedCode string
	}{
		{
			name:         "standard error code",
			expectedCode: "VALIDATION_ERROR",
		},
		{
			name:         "empty code",
			expectedCode: "",
		},
		{
			name:         "code with numbers",
			expectedCode: "ERROR_500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			err := webapp.NewHttpError(http.StatusBadRequest, tc.expectedCode, "Test description", nil)

			// Act
			code := err.Code()

			// Assert
			assert.Equal(t, tc.expectedCode, code)
		})
	}
}

func TestHttpError_Description(t *testing.T) {
	testCases := []struct {
		name                string
		expectedDescription string
	}{
		{
			name:                "standard description",
			expectedDescription: "The request contains invalid parameters",
		},
		{
			name:                "empty description",
			expectedDescription: "",
		},
		{
			name:                "long description",
			expectedDescription: "This is a very long description that contains multiple sentences. It should be handled correctly by the error implementation.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			err := webapp.NewHttpError(http.StatusBadRequest, "TEST_CODE", tc.expectedDescription, nil)

			// Act
			description := err.Description()

			// Assert
			assert.Equal(t, tc.expectedDescription, description)
		})
	}
}

func TestHttpError_Unwrap(t *testing.T) {
	testCases := []struct {
		name          string
		wrappedError  error
		expectsResult bool
	}{
		{
			name:          "with wrapped error",
			wrappedError:  errors.New("original error"),
			expectsResult: true,
		},
		{
			name:          "without wrapped error",
			wrappedError:  nil,
			expectsResult: false,
		},
		{
			name:          "with formatted error",
			wrappedError:  fmt.Errorf("formatted error: %w", errors.New("inner error")),
			expectsResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			err := webapp.NewHttpError(http.StatusInternalServerError, "TEST_CODE", "Test description", tc.wrappedError)

			// Act
			unwrapped := err.Unwrap()

			// Assert
			if tc.expectsResult {
				assert.NotNil(t, unwrapped)
				assert.Equal(t, tc.wrappedError, unwrapped)
			} else {
				assert.Nil(t, unwrapped)
			}
		})
	}
}

func TestHttpError_InterfaceCompliance(t *testing.T) {
	// Arrange
	err := webapp.NewHttpError(http.StatusBadRequest, "TEST_CODE", "Test description", nil)

	// Assert - verify that the error implements all expected interfaces
	assert.Implements(t, (*error)(nil), err)
	assert.Implements(t, (*webapp.BusinessError)(nil), err)
	assert.Implements(t, (*webapp.HttpError)(nil), err)
}

func TestNewHttpError(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		code        string
		description string
		wrapped     error
	}{
		{
			name:        "complete error",
			status:      http.StatusBadRequest,
			code:        "INVALID_REQUEST",
			description: "The request is invalid",
			wrapped:     errors.New("validation failed"),
		},
		{
			name:        "minimal error",
			status:      http.StatusNotFound,
			code:        "NOT_FOUND",
			description: "Resource not found",
			wrapped:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := webapp.NewHttpError(tc.status, tc.code, tc.description, tc.wrapped)

			// Assert
			require.NotNil(t, err)
			assert.Equal(t, tc.status, err.HttpStatus())
			assert.Equal(t, tc.code, err.Code())
			assert.Equal(t, tc.description, err.Description())
			assert.Equal(t, tc.wrapped, err.Unwrap())
		})
	}
}

func TestNewBadRequestError(t *testing.T) {
	testCases := []struct {
		name        string
		code        string
		description string
		wrapped     error
	}{
		{
			name:        "bad request with wrapped error",
			code:        "VALIDATION_FAILED",
			description: "Request validation failed",
			wrapped:     errors.New("field 'email' is required"),
		},
		{
			name:        "bad request without wrapped error",
			code:        "INVALID_FORMAT",
			description: "Invalid request format",
			wrapped:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := webapp.NewBadRequestError(tc.code, tc.description, tc.wrapped)

			// Assert
			require.NotNil(t, err)
			assert.Equal(t, http.StatusBadRequest, err.HttpStatus())
			assert.Equal(t, tc.code, err.Code())
			assert.Equal(t, tc.description, err.Description())
			assert.Equal(t, tc.wrapped, err.Unwrap())
		})
	}
}

func TestNewConflictError(t *testing.T) {
	testCases := []struct {
		name        string
		code        string
		description string
		wrapped     error
	}{
		{
			name:        "conflict with wrapped error",
			code:        "RESOURCE_EXISTS",
			description: "Resource already exists",
			wrapped:     errors.New("duplicate key violation"),
		},
		{
			name:        "conflict without wrapped error",
			code:        "CONCURRENT_MODIFICATION",
			description: "Resource was modified by another request",
			wrapped:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := webapp.NewConflictError(tc.code, tc.description, tc.wrapped)

			// Assert
			require.NotNil(t, err)
			assert.Equal(t, http.StatusConflict, err.HttpStatus())
			assert.Equal(t, tc.code, err.Code())
			assert.Equal(t, tc.description, err.Description())
			assert.Equal(t, tc.wrapped, err.Unwrap())
		})
	}
}

func TestNewNotFoundError(t *testing.T) {
	testCases := []struct {
		name        string
		code        string
		description string
	}{
		{
			name:        "user not found",
			code:        "USER_NOT_FOUND",
			description: "User with the specified ID does not exist",
		},
		{
			name:        "resource not found",
			code:        "RESOURCE_NOT_FOUND",
			description: "The requested resource could not be found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := webapp.NewNotFoundError(tc.code, tc.description)

			// Assert
			require.NotNil(t, err)
			assert.Equal(t, http.StatusNotFound, err.HttpStatus())
			assert.Equal(t, tc.code, err.Code())
			assert.Equal(t, tc.description, err.Description())
			assert.Nil(t, err.Unwrap()) // NotFoundError always has nil wrapped error
		})
	}
}

func TestNewInternalServerError(t *testing.T) {
	testCases := []struct {
		name        string
		code        string
		description string
		wrapped     error
	}{
		{
			name:        "internal error with wrapped error",
			code:        "DATABASE_CONNECTION_FAILED",
			description: "Failed to connect to database",
			wrapped:     errors.New("connection pool exhausted"),
		},
		{
			name:        "internal error without wrapped error",
			code:        "UNEXPECTED_ERROR",
			description: "An unexpected error occurred",
			wrapped:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := webapp.NewInternalServerError(tc.code, tc.description, tc.wrapped)

			// Assert
			require.NotNil(t, err)
			assert.Equal(t, http.StatusInternalServerError, err.HttpStatus())
			assert.Equal(t, tc.code, err.Code())
			assert.Equal(t, tc.description, err.Description())
			assert.Equal(t, tc.wrapped, err.Unwrap())
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Test that errors.Is and errors.As work properly with our HttpError implementation
	originalErr := errors.New("original error")
	wrappedErr := fmt.Errorf("wrapped: %w", originalErr)
	httpErr := webapp.NewInternalServerError("WRAPPED_ERROR", "Error with wrapped chain", wrappedErr)

	// Test errors.Is
	assert.ErrorIs(t, httpErr, originalErr)
	assert.ErrorIs(t, httpErr, wrappedErr)

	// Test errors.As
	var targetHttpErr webapp.HttpError
	assert.ErrorAs(t, httpErr, &targetHttpErr)
	assert.Equal(t, httpErr, targetHttpErr)

	var targetBusinessErr webapp.BusinessError
	assert.ErrorAs(t, httpErr, &targetBusinessErr)
}
