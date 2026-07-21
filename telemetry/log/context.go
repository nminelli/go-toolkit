// Package log uses ZAP fmt, which is a small wrapper around [Uber log package](https://godoc.org/go.uber.org/zap).
package log

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/nminelli/go-toolkit/telemetry/log/otelzap"
)

const (
	tagTraceID = "trace.id"
	tagSpanID  = "span.id"
)

type _loggerContextKey struct{}

// Context returns a new context with the given logger.
func Context(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, _loggerContextKey{}, logger)
}

// FromContext returns the logger from the context. If no logger is found, a new
// logger is created with the default configuration.
func FromContext(ctx context.Context) Logger {
	l, _ := ctx.Value(_loggerContextKey{}).(Logger)
	if l == nil {
		l = &logger{
			Logger: otelzap.Ctx(ctx),
		}
	}

	return l
}

// With returns a new context with a logger that includes the given fields.
func With(ctx context.Context, fields ...Field) context.Context {
	l := FromContext(ctx)
	return Context(ctx, l.With(fields...))
}

// Debug logs a message at DebugLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Debug(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Debug(msg, appendTracingFields(ctx, fields...)...)
}

// Error logs a message at ErrorLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Error(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Error(msg, appendTracingFields(ctx, fields...)...)
}

// Info logs a message at InfoLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Info(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Info(msg, appendTracingFields(ctx, fields...)...)
}

// Panic logs a message at PanicLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then panics, even if logging at PanicLevel is disabled.
func Panic(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Panic(msg, appendTracingFields(ctx, fields...)...)
}

// Fatal logs a message at FatalLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
//
// The logger then calls os.Exit(1), even if logging at FatalLevel is disabled.
// Note: This function should be used with caution, as it will terminate the program.
func Fatal(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Fatal(msg, appendTracingFields(ctx, fields...)...)
}

// Warn logs a message at WarnLevel. The message includes any fields passed
// at the log site, as well as any fields accumulated on the logger.
func Warn(ctx context.Context, msg string, fields ...Field) {
	FromContext(ctx).Warn(msg, appendTracingFields(ctx, fields...)...)
}

// appendTracingFields appends (if there is any) the required OpenTelemetry tracing fields.
//
// In order to enable log tracing, there must be a Span alive in the context passed in.
func appendTracingFields(ctx context.Context, fields ...Field) []Field {
	return append(fields, getTracingFields(ctx)...)
}

func getTracingFields(ctx context.Context) []Field {
	span := trace.SpanFromContext(ctx)
	sctx := span.SpanContext()
	if !sctx.IsValid() {
		return nil
	}

	return []Field{
		String(tagTraceID, sctx.TraceID().String()),
		String(tagSpanID, sctx.SpanID().String()),
	}
}
