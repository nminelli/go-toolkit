package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MFN-AISystems/go-toolkit/telemetry/internal/otelhttp"
)

func Test_Initialize_Success(t *testing.T) {
	// Set shorter timeouts for OTLP exporter
	os.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1")
	defer func() {
		os.Unsetenv("OTEL_EXPORTER_OTLP_TIMEOUT")
	}()

	a := assert.New(t)
	ctx := context.Background()

	cleanup, err := Init(ctx)
	a.NoError(err)
	a.NotNil(cleanup)

	cleanup()
}

func Test_NewOtelResource_WithoutEnvironmentVars(t *testing.T) {
	a := assert.New(t)
	ctx := context.Background()

	// Clear environment variables
	os.Unsetenv("APP_VERSION")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("AWS_LAMBDA_LOG_STREAM_NAME")

	res, err := newOtelResource(ctx)
	a.NoError(err)
	a.NotNil(res)
}

func Test_NewOtelResource_WithEnvironmentVars(t *testing.T) {
	a := assert.New(t)
	ctx := context.Background()

	// Set environment variables
	os.Setenv("APP_VERSION", "1.0.0")
	os.Setenv("ENVIRONMENT", "test")
	os.Setenv("AWS_LAMBDA_LOG_STREAM_NAME", "asdf12345")
	defer func() {
		os.Unsetenv("APP_VERSION")
		os.Unsetenv("ENVIRONMENT")
		os.Unsetenv("AWS_LAMBDA_LOG_STREAM_NAME")
	}()

	res, err := newOtelResource(ctx)
	a.NoError(err)
	a.NotNil(res)
}

func TestNewOTelRoundTripper(t *testing.T) {
	testCases := []struct {
		name             string
		base             http.RoundTripper
		tracerName       string
		tracerVersion    string
		setupMocks       func(ctx context.Context)
		runAssertions    func(t *testing.T, rt *otelhttp.OTelRoundTripper)
		expectedNotNil   bool
		expectedBaseName string
	}{
		{
			name:           "creates with custom base transport and custom tracer",
			base:           &http.Transport{},
			tracerName:     "my-service/http-client",
			tracerVersion:  "v1.2.3",
			setupMocks:     func(ctx context.Context) {},
			expectedNotNil: true,
			runAssertions: func(t *testing.T, rt *otelhttp.OTelRoundTripper) {
				assert.NotNil(t, rt)
			},
		},
		{
			name:           "uses default transport when base is nil",
			base:           nil,
			tracerName:     "my-service",
			tracerVersion:  "v2.0.0",
			setupMocks:     func(ctx context.Context) {},
			expectedNotNil: true,
			runAssertions: func(t *testing.T, rt *otelhttp.OTelRoundTripper) {
				assert.NotNil(t, rt)
			},
		},
		{
			name:           "uses default tracer name and version when empty",
			base:           &http.Transport{},
			tracerName:     "",
			tracerVersion:  "",
			setupMocks:     func(ctx context.Context) {},
			expectedNotNil: true,
			runAssertions: func(t *testing.T, rt *otelhttp.OTelRoundTripper) {
				assert.NotNil(t, rt)
			},
		},
		{
			name:           "uses default tracer name with empty tracer name but provided version",
			base:           nil,
			tracerName:     "",
			tracerVersion:  "v3.0.0",
			setupMocks:     func(ctx context.Context) {},
			expectedNotNil: true,
			runAssertions: func(t *testing.T, rt *otelhttp.OTelRoundTripper) {
				assert.NotNil(t, rt)
			},
		},
		{
			name:           "uses custom tracer name with default version",
			base:           nil,
			tracerName:     "",
			tracerVersion:  "0.0.0",
			setupMocks:     func(ctx context.Context) {},
			expectedNotNil: true,
			runAssertions: func(t *testing.T, rt *otelhttp.OTelRoundTripper) {
				assert.NotNil(t, rt)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tc.setupMocks(ctx)

			rt := NewOTelRoundTripper(tc.base, tc.tracerName, tc.tracerVersion)

			if tc.expectedNotNil {
				require.NotNil(t, rt)
			}

			tc.runAssertions(t, rt)
		})
	}
}

func TestNewOTelRoundTripper_Integration(t *testing.T) {
	// Set shorter timeouts for OTLP exporter
	os.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_TIMEOUT")

	// Initialize telemetry
	ctx := context.Background()
	cleanup, err := Init(ctx)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	testCases := []struct {
		name          string
		tracerName    string
		tracerVersion string
		setupServer   func() *httptest.Server
		makeRequest   func(t *testing.T, client *http.Client, serverURL string)
		assertions    func(t *testing.T)
	}{
		{
			name:          "successfully makes HTTP request with default tracer",
			tracerName:    "",
			tracerVersion: "",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"status":"ok"}`))
				}))
			},
			makeRequest: func(t *testing.T, client *http.Client, serverURL string) {
				resp, err := client.Get(serverURL)
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			},
			assertions: func(t *testing.T) {
				// Additional assertions can be added here
			},
		},
		{
			name:          "successfully makes HTTP request with custom tracer",
			tracerName:    "test-service/http-client",
			tracerVersion: "v1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			makeRequest: func(t *testing.T, client *http.Client, serverURL string) {
				req, err := http.NewRequest(http.MethodGet, serverURL, nil)
				require.NoError(t, err)

				resp, err := client.Do(req)
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			},
			assertions: func(t *testing.T) {
				// Additional assertions can be added here
			},
		},
		{
			name:          "handles server errors correctly",
			tracerName:    "test-service",
			tracerVersion: "v2.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error":"internal server error"}`))
				}))
			},
			makeRequest: func(t *testing.T, client *http.Client, serverURL string) {
				resp, err := client.Get(serverURL)
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
				resp.Body.Close()
			},
			assertions: func(t *testing.T) {
				// Error should be recorded in span
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test server
			server := tc.setupServer()
			defer server.Close()

			// Create instrumented HTTP client
			transport := NewOTelRoundTripper(nil, tc.tracerName, tc.tracerVersion)
			client := &http.Client{Transport: transport}

			// Make request
			tc.makeRequest(t, client, server.URL)

			// Run assertions
			tc.assertions(t)
		})
	}
}

func TestNewOTelRoundTripper_ChainedTransports(t *testing.T) {
	// Set shorter timeouts for OTLP exporter
	os.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_TIMEOUT")

	// Initialize telemetry
	ctx := context.Background()
	cleanup, err := Init(ctx)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	// Create a custom transport that adds a header
	customTransport := &customHeaderTransport{
		base:   http.DefaultTransport,
		header: "X-Custom-Header",
		value:  "custom-value",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom header is present
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Chain the OTel transport with the custom transport
	transport := NewOTelRoundTripper(customTransport, "chained-test", "v1.0.0")
	client := &http.Client{Transport: transport}

	// Make request
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// customHeaderTransport is a test helper that adds a custom header to requests
type customHeaderTransport struct {
	base   http.RoundTripper
	header string
	value  string
}

func (t *customHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set(t.header, t.value)
	return t.base.RoundTrip(req)
}

// Tests for pattern configuration functions

func TestWithAdditionalPatterns(t *testing.T) {
	t.Run("creates config with additional patterns", func(t *testing.T) {
		patterns := []PathPattern{
			{
				Matcher:     regexp.MustCompile(`^/special/[^/]+$`),
				Replacement: "/special/{id}",
			},
			{
				Matcher:     regexp.MustCompile(`^/custom/[^/]+$`),
				Replacement: "/custom/{id}",
			},
		}

		option := WithAdditionalPatterns(patterns)

		// Apply option to config
		config := &otelhttp.SpanNameConfig{}
		option(config)

		// Verify patterns were set
		assert.Len(t, config.AdditionalPatterns, 2)
		assert.NotNil(t, config.AdditionalPatterns[0].Matcher)
		assert.Equal(t, "/special/{id}", config.AdditionalPatterns[0].Replacement)
		assert.Equal(t, "/custom/{id}", config.AdditionalPatterns[1].Replacement)
	})

	t.Run("handles empty patterns slice", func(t *testing.T) {
		option := WithAdditionalPatterns([]PathPattern{})

		config := &otelhttp.SpanNameConfig{}
		option(config)

		assert.Len(t, config.AdditionalPatterns, 0)
	})
}

func TestWithCustomSanitizer(t *testing.T) {
	t.Run("sets custom sanitizer function", func(t *testing.T) {
		customFunc := func(method, path string) string {
			return method + " custom"
		}

		option := WithCustomSanitizer(customFunc)

		// Apply option to config
		config := &otelhttp.SpanNameConfig{}
		option(config)

		// Verify sanitizer was set
		assert.NotNil(t, config.CustomSanitizer)

		// Test the sanitizer
		result := config.CustomSanitizer("GET", "/test")
		assert.Equal(t, "GET custom", result)
	})

	t.Run("can set method-only sanitizer", func(t *testing.T) {
		option := WithCustomSanitizer(func(method, path string) string {
			return method
		})

		config := &otelhttp.SpanNameConfig{}
		option(config)

		assert.NotNil(t, config.CustomSanitizer)
		result := config.CustomSanitizer("POST", "/users/123")
		assert.Equal(t, "POST", result)
	})
}

func TestNewOTelRoundTripper_WithOptions(t *testing.T) {
	t.Run("creates with additional patterns option", func(t *testing.T) {
		patterns := []PathPattern{
			{
				Matcher:     regexp.MustCompile(`^/api/[^/]+$`),
				Replacement: "/api/{id}",
			},
		}

		rt := NewOTelRoundTripper(nil, "test-service", "v1.0.0",
			WithAdditionalPatterns(patterns))

		assert.NotNil(t, rt)
	})

	t.Run("creates with custom sanitizer option", func(t *testing.T) {
		rt := NewOTelRoundTripper(nil, "test-service", "v1.0.0",
			WithCustomSanitizer(func(method, path string) string {
				return method
			}))

		assert.NotNil(t, rt)
	})

	t.Run("creates with multiple options", func(t *testing.T) {
		patterns := []PathPattern{
			{
				Matcher:     regexp.MustCompile(`^/special/[^/]+$`),
				Replacement: "/special/{id}",
			},
		}

		// Note: Custom sanitizer will override patterns, but we can still pass both
		rt := NewOTelRoundTripper(nil, "test-service", "v1.0.0",
			WithAdditionalPatterns(patterns),
			WithCustomSanitizer(func(method, path string) string {
				return method + " " + path
			}))

		assert.NotNil(t, rt)
	})

	t.Run("creates without options", func(t *testing.T) {
		rt := NewOTelRoundTripper(nil, "test-service", "v1.0.0")
		assert.NotNil(t, rt)
	})
}

func TestNewOTelRoundTripper_Integration_WithPatterns(t *testing.T) {
	// Set shorter timeouts for OTLP exporter
	os.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_TIMEOUT")

	// Initialize telemetry
	ctx := context.Background()
	cleanup, err := Init(ctx)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	defer cleanup()

	t.Run("uses additional patterns in real request", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create transport with additional patterns
		patterns := []PathPattern{
			{
				Matcher:     regexp.MustCompile(`^/custom/[^/]+$`),
				Replacement: "/custom/{id}",
			},
		}

		transport := NewOTelRoundTripper(nil, "pattern-test", "v1.0.0",
			WithAdditionalPatterns(patterns))
		client := &http.Client{Transport: transport}

		// Make request (this will use the patterns internally)
		resp, err := client.Get(server.URL + "/custom/abc123")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("uses custom sanitizer in real request", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create transport with custom sanitizer
		transport := NewOTelRoundTripper(nil, "sanitizer-test", "v1.0.0",
			WithCustomSanitizer(func(method, path string) string {
				return method // Use method-only
			}))
		client := &http.Client{Transport: transport}

		// Make request
		resp, err := client.Get(server.URL + "/users/550e8400-e29b-41d4-a716-446655440000")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})
}
