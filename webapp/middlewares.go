package webapp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/nminelli/go-toolkit/httprouter"
	"github.com/nminelli/go-toolkit/telemetry/log"
	"github.com/nminelli/go-toolkit/telemetry/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.0"
	"go.opentelemetry.io/otel/semconv/v1.37.0/httpconv"
	"go.opentelemetry.io/otel/trace"
)

func ignoredRoute(path string) bool {
	return slices.Contains(_telemetryIgnoreRoutes, path)
}

// newCompressor returns a middleware that compresses response body of a given content type to a data format based
// on Accept-Encoding request header. It uses the _defaultCompressionLevel.
//
// NOTE: if you don't use web.RespondJSON to marshal the body into the writer,
// make sure to set the Content-Type header on your response otherwise this middleware will not compress the response body.
func newCompressor() func(next http.Handler) http.Handler {
	c := chiMiddleware.NewCompressor(_defaultCompressionLevel)
	return func(next http.Handler) http.Handler {
		return c.Handler(next)
	}
}

// Telemetry middleware simplifies tracing of incoming web requests by
// initiating a new Span and composing the request context with it.
func telemetryMiddleware(appName string, logger log.Logger) func(next http.Handler) http.Handler {
	meter := otel.Meter(appName, metric.WithInstrumentationVersion(Version()))
	httpServerRequestDuration, err := httpconv.NewServerRequestDuration(meter, metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10))
	if err != nil {
		panic(fmt.Sprintf("could not create http server request duration metric: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ignoredRoute(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Start with URL path as initial route pattern
			initialRoutePattern := r.URL.Path
			txName := fmt.Sprintf("%s %s", r.Method, initialRoutePattern)

			// Create a new span for the incoming request
			ctx, span := otel.Tracer(appName, trace.WithInstrumentationVersion(Version())).
				Start(r.Context(), txName, trace.WithSpanKind(trace.SpanKindServer))

			ctx = log.Context(ctx, logger)
			r2 := r.WithContext(ctx)

			// Wrap the http.ResponseWriter with a proxy for later response
			// inspection.
			w2 := chiMiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			defer func() {
				end := time.Now()
				elapsed := end.Sub(start)
				defer span.End(trace.WithTimestamp(end))

				var err error
				var panicked bool
				var statusCode int
				if recovered := recover(); recovered != nil {
					panicked = true
					panicInfo := FormatPanic(recovered)
					err = panicInfo.Error
					// TODO: Log lib should handle automatically OTEL fields
					log.Error(r2.Context(), panicInfo.Message, log.Err(panicInfo.Error), log.String(string(semconv.ExceptionStacktraceKey), panicInfo.StackTrace))

					JSONError(r2.Context(), w2, NewInternalServerError("COB-001", "COB-Desc", httprouter.NewError(statusCode, panicInfo.Message)))
				}

				if !panicked {
					statusCode = w2.Status()
				}

				// Get the actual route pattern after request processing when route context is available
				routePattern := chi.RouteContext(r2.Context()).RoutePattern()
				if strings.TrimSpace(routePattern) == "" {
					routePattern = r2.URL.Path
				}

				// Update span name with the correct route pattern if it changed
				actualTxName := fmt.Sprintf("%s %s", r2.Method, routePattern)

				span.SetAttributes(
					semconv.HTTPRoute(routePattern),
					semconv.HTTPRequestMethodKey.String(r2.Method),
				)

				// Update span name if it changed
				if actualTxName != txName {
					span.SetName(actualTxName)
				}

				if statusCode >= 500 {
					if err == nil {
						err = errors.New(http.StatusText(statusCode))
					}

					tracing.RecordError(r2.Context(), err)
				}

				recordRequest(r2, statusCode, elapsed, r2.Method, routePattern, httpServerRequestDuration)
			}()

			next.ServeHTTP(w2, r2)
		})
	}
}

// headerForwarder decorates a request context with the value of certain headers
// in order to allow transport.HTTPRequester to use those headers in outgoing requests.
func headerForwarder(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

		r2 := r.WithContext(ctx)
		next.ServeHTTP(w, r2)
	})
}

// log decorates the request context with the given logger, accessible via
// the go-core log methods with context.
func logMiddleware(config loggingConfig) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			if ignoredRoute(path) {
				next.ServeHTTP(w, r)
				return
			}

			logFields := []log.Field{
				log.String("method", r.Method),
				log.String("path", path),
				log.String("remoteaddr", r.RemoteAddr),
			}

			if config.LogRequests {
				// Read the entire request body upfront for logging
				origBody := r.Body
				defer origBody.Close()

				bodyBytes, err := io.ReadAll(origBody)
				if err != nil {
					// If we can't read the body, log the error but continue
					logFields = append(logFields, log.String("request.body", err.Error()))
				} else {
					// Store the body for logging
					logFields = append(logFields, log.String("request.body", string(bodyBytes)))

					// Create a new reader with the body data for the handlers to use
					r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
			}

			if r.URL.RawQuery != "" {
				path = fmt.Sprintf("%s?%s", path, r.URL.RawQuery)
			}

			log.Info(r.Context(), fmt.Sprintf("%s %s started", r.Method, path), logFields...)

			// Always wrap the response writer to capture status code and response body
			var respBuf *bytes.Buffer
			if config.LogResponses {
				respBuf = &bytes.Buffer{}
			}

			respWriter := &responseWriterWrapper{
				ResponseWriter: w,
				buffer:         respBuf,
				statusCode:     200, // Default status code
			}

			next.ServeHTTP(respWriter, r)

			// Prepare completion log fields
			completionLogFields := []log.Field{
				log.String("method", r.Method),
				log.String("path", path),
				log.Int("status_code", respWriter.statusCode),
			}

			// Log response body after request is processed
			if config.LogResponses && respBuf != nil {
				completionLogFields = append(completionLogFields, log.String("response_body", respBuf.String()))
			}

			// Log completion message
			completionMsg := fmt.Sprintf("%s %s completed", r.Method, path)

			// Add additional logging for error status codes (>399)
			if respWriter.statusCode > 399 {
				completionLogFields = append(completionLogFields,
					log.String("error_category", "http_error"),
					log.String("status_class", fmt.Sprintf("%dxx", respWriter.statusCode/100)),
				)
				log.Error(r.Context(), fmt.Sprintf("%s with error status %d", completionMsg, respWriter.statusCode), completionLogFields...)
			} else {
				log.Info(r.Context(), completionMsg, completionLogFields...)
			}
		})
	}
}
