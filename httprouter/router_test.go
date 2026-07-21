package httprouter_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MFN-AISystems/go-toolkit/httprouter"
	"github.com/MFN-AISystems/go-toolkit/httprouter/middleware"
)

func TestNewConfigHandlers(t *testing.T) {
	tests := []struct {
		name           string
		profilerActive bool
		inputURL       string
		wantCode       int
		wantMsg        string
	}{
		{
			name:     "liveness",
			inputURL: "/health/live",
			wantCode: http.StatusNoContent,
			wantMsg:  "",
		},
		{
			name:     "readiness",
			inputURL: "/health/ready",
			wantCode: http.StatusNoContent,
			wantMsg:  "",
		},
		{
			name:     "not found handler",
			inputURL: "/notfound",
			wantCode: http.StatusNotFound,
			wantMsg:  "handler not found",
		},
		{
			name:     "profiler is not active",
			inputURL: "/debug",
			wantCode: http.StatusNotFound,
		},
		{
			name:           "profiler is active",
			profilerActive: true,
			inputURL:       "/debug",
			wantCode:       http.StatusOK,
		},
		{
			name:     "with middleware",
			inputURL: "/",
			wantCode: http.StatusNotFound,
			wantMsg:  "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			livenessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			readinessHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			notFoundHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				writeLen, err := w.Write([]byte("handler not found"))
				assert.NoError(t, err)
				assert.EqualValues(t, 17, writeLen)
			})

			mw := func(f http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					f.ServeHTTP(w, r)
				})
			}

			r := httprouter.New(
				httprouter.WithGlobalMiddlewares(mw),
				httprouter.WithHealthCheckLivenessHandler(livenessHandler),
				httprouter.WithHealthCheckReadinessHandler(readinessHandler),
				httprouter.WithNotFoundHandler(notFoundHandler),
				httprouter.WithEnableProfiler(tc.profilerActive),
			)

			server := httptest.NewServer(r)
			defer server.Close()

			res, err := http.Get(fmt.Sprintf("%s%s", server.URL, tc.inputURL))
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tc.wantCode, res.StatusCode)

			if tc.wantMsg != "" {
				b, err := io.ReadAll(res.Body)
				assert.Nil(t, err)
				assert.Equal(t, tc.wantMsg, string(b))
			}
		})
	}
}

func TestRouterMethod(t *testing.T) {
	tests := []struct {
		name     string
		shortcut func(r *httprouter.Router, path string, handler http.Handler)
		method   string
	}{
		{
			name:     "get",
			shortcut: (*httprouter.Router).Get,
			method:   http.MethodGet,
		},
		{
			name:     "head",
			shortcut: (*httprouter.Router).Head,
			method:   http.MethodHead,
		},
		{
			name:     "options",
			shortcut: (*httprouter.Router).Options,
			method:   http.MethodOptions,
		},
		{
			name:     "post",
			shortcut: (*httprouter.Router).Post,
			method:   http.MethodPost,
		},
		{
			name:     "put",
			shortcut: (*httprouter.Router).Put,
			method:   http.MethodPut,
		},
		{
			name:     "patch",
			shortcut: (*httprouter.Router).Patch,
			method:   http.MethodPatch,
		},
		{
			name:     "delete",
			shortcut: (*httprouter.Router).Delete,
			method:   http.MethodDelete,
		},
		{
			name:     "trace",
			shortcut: (*httprouter.Router).Trace,
			method:   http.MethodTrace,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httprouter.New()

			server := httptest.NewServer(r)
			defer server.Close()

			test.shortcut(r, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req, err := http.NewRequest(test.method, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			client := http.Client{}
			resp, err := client.Do(req)
			assert.Nil(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestRouterRoutes(t *testing.T) {
	r := httprouter.New()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/test", h)

	routes, err := r.Routes()

	assert.Nil(t, err)
	assert.Len(t, routes, 1)
	assert.Equal(t, "/test", routes[0].Route)
	assert.Equal(t, "GET", routes[0].Method)
	assert.Len(t, routes[0].Middlewares, 0)
	assert.NotNil(t, routes[0].Handler)
}

func TestHeaderForwarder(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "with trace context",
			headers: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				"tracestate":  "rojo=00f067aa0ba902b7",
			},
		},
		{
			name: "with baggage",
			headers: map[string]string{
				"baggage": "userId=alice,serverNode=DF%2028",
			},
		},
		{
			name: "with both trace and baggage",
			headers: map[string]string{
				"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
				"baggage":     "userId=alice",
			},
		},
		{
			name:    "without headers",
			headers: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := middleware.HeaderForwarder

			// Create test handler that verifies context
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify context is not nil and has been processed
				assert.NotNil(t, r.Context())
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("ok"))
			}))

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(w, req)

			// Verify
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "ok", w.Body.String())
		})
	}
}

func TestRouterAdvancedFeatures(t *testing.T) {
	t.Run("With method for inline middlewares", func(t *testing.T) {
		router := httprouter.New()

		// Test middleware
		middlewareCalled := false
		testMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				middlewareCalled = true
				next.ServeHTTP(w, r)
			})
		}

		// Create router with middleware
		routerWithMiddleware := router.With(testMiddleware)

		routerWithMiddleware.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))

		server := httptest.NewServer(routerWithMiddleware)
		defer server.Close()

		resp, err := http.Get(server.URL + "/test")
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, middlewareCalled)
	})

	t.Run("Group method for grouped routes", func(t *testing.T) {
		router := httprouter.New()

		groupMiddlewareCalled := false
		groupMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				groupMiddlewareCalled = true
				next.ServeHTTP(w, r)
			})
		}

		// Create a group with middleware
		groupRouter := router.Group(func(r httprouter.Router) {
			r.Use(groupMiddleware)
			r.Get("/grouped", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"group":"test"}`))
			}))
		})

		server := httptest.NewServer(groupRouter)
		defer server.Close()

		resp, err := http.Get(server.URL + "/grouped")
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, groupMiddlewareCalled)
	})

	t.Run("Route method for subroutes", func(t *testing.T) {
		router := httprouter.New()

		// Add a main route first
		router.Get("/main", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"main":"route"}`))
		}))

		// Create subroute
		subRouter := router.Route("/api", func(r httprouter.Router) {
			r.Get("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"users":"list"}`))
			}))
			r.Post("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"user":"created"}`))
			}))
		})

		server := httptest.NewServer(router)
		defer server.Close()

		// Test main route
		resp, err := http.Get(server.URL + "/main")
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Test subroute GET
		resp, err = http.Get(server.URL + "/api/users")
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Test subroute POST
		resp, err = http.Post(server.URL+"/api/users", "application/json", nil)
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Verify subrouter was returned
		assert.NotNil(t, subRouter)
	})

	t.Run("Mount method for mounting handlers", func(t *testing.T) {
		router := httprouter.New()

		// Create a simple handler to mount (converted to httprouter.Handler)
		mountedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("mounted"))
		})

		// Mount the handler
		router.Mount("/mounted", mountedHandler)

		server := httptest.NewServer(router)
		defer server.Close()

		resp, err := http.Get(server.URL + "/mounted")
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, "mounted", string(body))
	})

	t.Run("Connect method", func(t *testing.T) {
		router := httprouter.New()

		router.Connect("/connect", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"method":"connect"}`))
		}))

		server := httptest.NewServer(router)
		defer server.Close()

		client := &http.Client{}
		req, err := http.NewRequest("CONNECT", server.URL+"/connect", nil)
		assert.NoError(t, err)

		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestRouteMethodPanicScenario(t *testing.T) {
	router := httprouter.New()

	// Test that Route panics when fn is nil
	assert.Panics(t, func() {
		router.Route("/panic", nil)
	})
}
