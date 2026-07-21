package tracing

import (
	"context"
	"fmt"
	"reflect"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.31.0"
	"go.opentelemetry.io/otel/trace"
)

// RecordError records an error in the current span and marks it as an error.
func RecordError(ctx context.Context, err error) {
	span := SpanFromContext(ctx)
	if span != nil {
		attrs := []attribute.KeyValue{
			semconv.ExceptionMessage(err.Error()),
			semconv.ExceptionType(reflect.TypeOf(err).String()),
			semconv.ErrorTypeKey.String(fmt.Sprintf("%T", err)),
		}

		if st, ok := err.(stackTracer); ok {
			stackTrace := &stackTrace{st}
			attrs = append(attrs, semconv.ExceptionStacktrace(stackTrace.String()))
		} else if sst, ok := err.(strStackTracer); ok {
			attrs = append(attrs, semconv.ExceptionStacktrace(sst.StackTrace()))
		}

		span.AddEvent(semconv.ExceptionEventName, trace.WithAttributes(attrs...))
		span.SetAttributes(attrs...)
		span.SetStatus(codes.Error, err.Error())
	}
}

// AddAttribute adds an attribute to the current span.
func AddAttribute(ctx context.Context, attribs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attribs...)
	}
}

// SpanFromContext retrieves the current span from the context.
// If no span is found, it returns a new span from a background context.
func SpanFromContext(ctx context.Context) trace.Span {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return trace.SpanFromContext(context.Background())
	}
	return span
}

// GetTextMapPropagator returns the global TextMapPropagator used for context propagation.
// This propagator is used to extract and inject context into HTTP headers or other text-based formats.
// It is typically used in distributed tracing to propagate trace context across service boundaries.
// The default propagator is set to the W3C Trace Context and Baggage propagators.
func GetTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}
