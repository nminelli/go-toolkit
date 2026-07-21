package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// BreakerValidator is a function that determines if a status code written to a
// client by a circuit breaking Handler should count as a success or failure.
// The DefaultBreakerValidator can be used in most situations.
type BreakerValidator func(int) bool

// DefaultBreakerValidator considers any status code less than 500 to be a
// success, from the perspective of a server. All other codes are failures.
func DefaultBreakerValidator(code int) bool { return code < 500 }

// CircuitBreaker represents a circuit breaker that can be used to control the flow of requests.
type CircuitBreaker interface {
	// Allow checks if a request is allowed to proceed.
	// Returns true if the request is allowed, false otherwise.
	Allow() bool

	// Success signals a successful request to the circuit breaker.
	// This can be used to update internal metrics or state.
	Success()

	// Failure signals a failed request to the circuit breaker.
	// This can be used to update internal metrics or state.
	Failure()
}

// Breaker produces a middleware that's governed by the passed Breaker and
// BreakerValidator. Responses written by the next Handler whose status codes
// fail the validator signal failures to the breaker. Once the breaker opens,
// incoming requests are terminated before being answered with HTTP 503.
func Breaker(cb CircuitBreaker, validator BreakerValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cb.Allow() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w2 := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(w2, r)

			if validator(w2.Status()) {
				cb.Success()
			} else {
				cb.Failure()
			}
		})
	}
}
