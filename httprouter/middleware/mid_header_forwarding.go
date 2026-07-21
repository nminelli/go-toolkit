package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel/propagation"
)

// HeaderForwarder decorates a request context with the value of certain headers
// in order to allow transport.HTTPRequester to use those headers in outgoing requests.
func HeaderForwarder(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

		r2 := r.WithContext(ctx)
		next.ServeHTTP(w, r2)
	})
}
