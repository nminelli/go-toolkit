package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nminelli/go-toolkit/telemetry"
)

func init() {
	_, _ = telemetry.Init(context.Background())
}

func TestClientID_HeaderSource(t *testing.T) {
	testCases := []struct {
		name           string
		clientIDHeader string
		expectedStatus int
		expectedValue  string
		setupMocks     func(ctx context.Context)
		runAssertions  func(t *testing.T, rr *httptest.ResponseRecorder, err error)
	}{
		{
			name:           "valid client ID header",
			clientIDHeader: "test-client-123",
			expectedStatus: http.StatusOK,
			expectedValue:  "test-client-123",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rr.Code)
			},
		},
		{
			name:           "missing client ID header",
			clientIDHeader: "",
			expectedStatus: http.StatusUnauthorized,
			expectedValue:  "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.setupMocks(ctx)

			// Create a test handler that checks the client ID in context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				clientID, exists := getClientIDFromContext(r.Context())
				if exists {
					assert.Equal(t, tc.expectedValue, clientID)
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create the middleware
			middleware := ClientID(SourceHeader)
			handler := middleware(testHandler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.clientIDHeader != "" {
				req.Header.Set("client_id", tc.clientIDHeader)
			}

			// Record the response
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			tc.runAssertions(t, rr, nil)
		})
	}
}

func TestClientID_JWTSource(t *testing.T) {
	testCases := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedValue  string
		setupMocks     func(ctx context.Context)
		runAssertions  func(t *testing.T, rr *httptest.ResponseRecorder, err error)
	}{
		{
			name:           "valid JWT token",
			authHeader:     createValidJWTHeader("test-client-456"),
			expectedStatus: http.StatusOK,
			expectedValue:  "test-client-456",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rr.Code)
			},
		},
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedValue:  "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
		{
			name:           "invalid JWT format",
			authHeader:     "Bearer invalid.jwt.token",
			expectedStatus: http.StatusUnauthorized,
			expectedValue:  "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
		{
			name:           "invalid client type",
			authHeader:     createJWTWithClientType("invalid_type"),
			expectedStatus: http.StatusUnauthorized,
			expectedValue:  "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
		{
			name:           "missing linked clients",
			authHeader:     createJWTWithoutLinkedClients(),
			expectedStatus: http.StatusUnauthorized,
			expectedValue:  "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
		{
			name:           "missing client type",
			authHeader:     createJWTWithoutClientType(),
			expectedStatus: http.StatusUnauthorized,
			expectedValue:  "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusUnauthorized, rr.Code)
			},
		},
		{
			name:           "client ID with pipe separator",
			authHeader:     createValidJWTHeaderWithPipe("client-123|additional-data"),
			expectedStatus: http.StatusOK,
			expectedValue:  "client-123",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, rr *httptest.ResponseRecorder, err error) {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rr.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.setupMocks(ctx)

			// Create a test handler that checks the client ID in context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				clientID, exists := getClientIDFromContext(r.Context())
				if exists {
					assert.Equal(t, tc.expectedValue, clientID)
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create the middleware with JWT source
			middleware := ClientID(SourceJWT)
			handler := middleware(testHandler)

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			// Record the response
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			tc.runAssertions(t, rr, nil)
		})
	}
}

func TestRetrieveClientIDFromHeader(t *testing.T) {
	testCases := []struct {
		name           string
		clientIDHeader string
		expectedResult string
		setupMocks     func(ctx context.Context)
		runAssertions  func(t *testing.T, result string, err error)
	}{
		{
			name:           "valid client ID",
			clientIDHeader: "test-client-123",
			expectedResult: "test-client-123",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test-client-123", result)
			},
		},
		{
			name:           "empty client ID",
			clientIDHeader: "",
			expectedResult: "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "client_id header missing")
				assert.Equal(t, "", result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.setupMocks(ctx)

			req := httptest.NewRequest("GET", "/test", nil)
			if tc.clientIDHeader != "" {
				req.Header.Set("client_id", tc.clientIDHeader)
			}

			result, err := retrieveClientIDFromHeader(req)
			tc.runAssertions(t, result, err)
		})
	}
}

func TestRetrieveClientIDFromJWT(t *testing.T) {
	testCases := []struct {
		name           string
		authHeader     string
		expectedResult string
		errorContains  string
		setupMocks     func(ctx context.Context)
		runAssertions  func(t *testing.T, result string, err error)
	}{
		{
			name:           "valid JWT",
			authHeader:     createValidJWTHeader("test-client-789"),
			expectedResult: "test-client-789",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test-client-789", result)
			},
		},
		{
			name:          "missing authorization header",
			authHeader:    "",
			errorContains: "authorization header is missing",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "authorization header is missing")
			},
		},
		{
			name:          "invalid JWT format",
			authHeader:    "Bearer invalid.jwt",
			errorContains: "invalid token",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid token")
			},
		},
		{
			name:          "invalid client type",
			authHeader:    createJWTWithClientType("invalid_type"),
			errorContains: "invalid token type",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid token type")
			},
		},
		{
			name:          "missing linked clients",
			authHeader:    createJWTWithoutLinkedClients(),
			errorContains: "client ID is missing",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "client ID is missing")
			},
		},
		{
			name:          "missing client type",
			authHeader:    createJWTWithoutClientType(),
			errorContains: "invalid token",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid token")
			},
		},
		{
			name:           "client ID with pipe separator",
			authHeader:     createValidJWTHeaderWithPipe("client-456|extra-data"),
			expectedResult: "client-456",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "client-456", result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.setupMocks(ctx)

			req := httptest.NewRequest("GET", "/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			result, err := retrieveClientIDFromJWT(req)
			tc.runAssertions(t, result, err)
		})
	}
}

func TestExtractClaimsFromJWTHeader(t *testing.T) {
	testCases := []struct {
		name          string
		authHeader    string
		errorContains string
		setupMocks    func(ctx context.Context)
		runAssertions func(t *testing.T, result cobreClaims, err error)
	}{
		{
			name:       "valid JWT header",
			authHeader: createValidJWTHeader("test-client"),
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result cobreClaims, err error) {
				assert.NoError(t, err)
				assert.NotEqual(t, cobreClaims{}, result)
			},
		},
		{
			name:       "valid JWT header without Bearer prefix",
			authHeader: createValidJWTHeader("test-client")[len("Bearer "):],
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result cobreClaims, err error) {
				assert.NoError(t, err)
				assert.NotEqual(t, cobreClaims{}, result)
			},
		},
		{
			name:          "invalid base64 encoding",
			authHeader:    "Bearer header.invalid-base64.signature",
			errorContains: "invalid JWT format",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result cobreClaims, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid JWT format")
				assert.Equal(t, cobreClaims{}, result)
			},
		},
		{
			name:          "invalid JSON in payload",
			authHeader:    fmt.Sprintf("Bearer header.%s.signature", base64.RawStdEncoding.EncodeToString([]byte("invalid-json"))),
			errorContains: "invalid JWT format",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result cobreClaims, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid JWT format")
				assert.Equal(t, cobreClaims{}, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.setupMocks(ctx)

			result, err := extractClaimsFromJWTHeader(tc.authHeader)
			tc.runAssertions(t, result, err)
		})
	}
}

func TestGetClientIDFromContext(t *testing.T) {
	testCases := []struct {
		name           string
		setupContext   func() context.Context
		expectedResult string
		setupMocks     func(ctx context.Context)
		runAssertions  func(t *testing.T, result string, err error)
	}{
		{
			name: "context with client ID",
			setupContext: func() context.Context {
				return AddClientIDToContext(context.Background(), "test-client-123")
			},
			expectedResult: "test-client-123",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "test-client-123", result)
			},
		},
		{
			name: "context without client ID",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectedResult: "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "", result)
			},
		},
		{
			name: "context with wrong type value",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), clientIDKey{}, 12345)
			},
			expectedResult: "",
			setupMocks: func(ctx context.Context) {
				// No mocks needed for this test
			},
			runAssertions: func(t *testing.T, result string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "", result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.setupContext()
			tc.setupMocks(ctx)

			result := GetClientIDFromContext(ctx)
			tc.runAssertions(t, result, nil)
		})
	}
}

// Helper functions to create test JWTs

func createValidJWTHeader(clientID string) string {
	payload := map[string]interface{}{
		"custom:type":           "api_user",
		"custom:linked_clients": clientID,
	}
	return createJWTHeader(payload)
}

func createValidJWTHeaderWithPipe(clientData string) string {
	payload := map[string]interface{}{
		"custom:type":           "api_user",
		"custom:linked_clients": clientData,
	}
	return createJWTHeader(payload)
}

func createJWTWithClientType(clientType string) string {
	payload := map[string]interface{}{
		"custom:type":           clientType,
		"custom:linked_clients": "test-client",
	}
	return createJWTHeader(payload)
}

func createJWTWithoutLinkedClients() string {
	payload := map[string]interface{}{
		"custom:type": "api_user",
	}
	return createJWTHeader(payload)
}

func createJWTWithoutClientType() string {
	payload := map[string]interface{}{
		"custom:linked_clients": "test-client",
	}
	return createJWTHeader(payload)
}

func createJWTHeader(payload map[string]interface{}) string {
	// Create a simple header
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerBytes, _ := json.Marshal(header)
	payloadBytes, _ := json.Marshal(payload)

	headerEncoded := base64.RawStdEncoding.EncodeToString(headerBytes)
	payloadEncoded := base64.RawStdEncoding.EncodeToString(payloadBytes)

	// Simple signature (not cryptographically valid, just for testing)
	signature := "test-signature"

	return fmt.Sprintf("Bearer %s.%s.%s", headerEncoded, payloadEncoded, signature)
}

func getClientIDFromContext(ctx context.Context) (string, bool) {
	clientID, ok := ctx.Value(clientIDKey{}).(string)
	return clientID, ok
}
