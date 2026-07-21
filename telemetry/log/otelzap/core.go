package otelzap

import (
	"fmt"
	"reflect"
	"runtime"

	otel "github.com/agoda-com/opentelemetry-logs-go/logs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	instrumentationName = "github.com/nminelli/go-toolkit/telemetry/log"
)

// This class provide interface for OTLP logger
type otlpCore struct {
	logger otel.Logger

	fields       []zapcore.Field
	levelEnabler zapcore.LevelEnabler
}

var instrumentationScope = instrumentation.Scope{
	Name:      instrumentationName,
	Version:   Version(),
	SchemaURL: semconv.SchemaURL,
}

func (c *otlpCore) Enabled(level zapcore.Level) bool {
	return c.levelEnabler.Enabled(level)
}

func (c *otlpCore) With(f []zapcore.Field) zapcore.Core {
	fields := c.fields
	fields = append(fields, f...)

	return &otlpCore{
		logger:       c.logger,
		fields:       fields,
		levelEnabler: c.levelEnabler,
	}
}

// Check OTLP zap extension method to check if logger is enabled
func (c *otlpCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *otlpCore) Sync() error {
	return nil
}

func (c *otlpCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	attributes, spanCtx := c.collectAttributes(ent, fields)
	attributes = c.addExceptionInfo(attributes, fields)

	record := c.createLogRecord(ent, attributes, spanCtx)
	c.logger.Emit(record)
	return nil
}

func (c *otlpCore) collectAttributes(ent zapcore.Entry, fields []zapcore.Field) ([]attribute.KeyValue, *trace.SpanContext) {
	attributes, spanCtx := c.processCommonFields()
	return c.processLogFields(ent, attributes, fields), spanCtx
}

func (c *otlpCore) processCommonFields() ([]attribute.KeyValue, *trace.SpanContext) {
	var attributes []attribute.KeyValue
	var spanCtx *trace.SpanContext

	for _, s := range c.fields {
		if s.Key == "context" {
			if ctxValue, ok := s.Interface.(trace.SpanContext); ok {
				spanCtx = &ctxValue
				continue
			}
		}
		attributes = append(attributes, otelAttribute(s)...)
	}
	return attributes, spanCtx
}

func (c *otlpCore) processLogFields(ent zapcore.Entry, attributes []attribute.KeyValue, fields []zapcore.Field) []attribute.KeyValue {
	hasLevel := false
	for _, s := range fields {
		hasLevel = hasLevel || s.Key == "level"
		// Skip error fields as they are handled by addExceptionInfo
		if s.Type == zapcore.ErrorType {
			continue
		}
		attributes = append(attributes, otelAttribute(s)...)
	}

	if !hasLevel {
		attributes = append(attributes, otelAttribute(zap.String("level", ent.Level.String()))...)
	}
	return attributes
}

func (c *otlpCore) addExceptionInfo(attributes []attribute.KeyValue, fields []zapcore.Field) []attribute.KeyValue {
	// Find error fields and extract exception information from them
	for _, field := range fields {
		if field.Type == zapcore.ErrorType && field.Interface != nil {
			if err, ok := field.Interface.(error); ok {
				// Extract exception type (error type name)
				exceptionType := c.getErrorType(err)
				if exceptionType != "" {
					attributes = append(attributes, semconv.ExceptionType(exceptionType))
				}

				// Extract exception message
				if err.Error() != "" {
					attributes = append(attributes, attribute.String("exception.message", err.Error()))
				}

				// Extract stacktrace if available
				stackTrace := c.getErrorStackTrace(err)
				if stackTrace != "" {
					attributes = append(attributes, semconv.ExceptionStacktrace(stackTrace))
				}

				// Only process the first error field found
				break
			}
		}
	}

	return attributes
}

func (c *otlpCore) createLogRecord(ent zapcore.Entry, attributes []attribute.KeyValue, spanCtx *trace.SpanContext) otel.LogRecord {
	severityString := ent.Level.String()
	severity := otelLevel(ent.Level)
	traceID, spanID, traceFlags := c.extractTraceInfo(spanCtx)

	lr := otel.LogRecordConfig{
		Timestamp:            &ent.Time,
		ObservedTimestamp:    ent.Time,
		TraceId:              traceID,
		SpanId:               spanID,
		TraceFlags:           traceFlags,
		SeverityText:         &severityString,
		SeverityNumber:       &severity,
		BodyAny:              &ent.Message,
		Resource:             nil,
		InstrumentationScope: &instrumentationScope,
		Attributes:           &attributes,
	}

	return otel.NewLogRecord(lr)
}

func (c *otlpCore) extractTraceInfo(spanCtx *trace.SpanContext) (*trace.TraceID, *trace.SpanID, *trace.TraceFlags) {
	if spanCtx == nil {
		return nil, nil, nil
	}

	tid := spanCtx.TraceID()
	sid := spanCtx.SpanID()
	tf := spanCtx.TraceFlags()
	return &tid, &sid, &tf
}

// getErrorType extracts the type name from an error
func (c *otlpCore) getErrorType(err error) string {
	if err == nil {
		return ""
	}

	// Get the actual type of the error
	errorType := reflect.TypeOf(err)

	// Handle pointer types
	if errorType.Kind() == reflect.Ptr {
		errorType = errorType.Elem()
	}

	// Return the full type name including package
	if errorType.PkgPath() != "" {
		return fmt.Sprintf("%s.%s", errorType.PkgPath(), errorType.Name())
	}

	return errorType.Name()
}

// getErrorStackTrace attempts to extract stack trace from error
func (c *otlpCore) getErrorStackTrace(err error) string {
	if err == nil {
		return ""
	}

	// Try to extract stack trace using different common interfaces

	// Check if error implements a StackTrace method (like pkg/errors)
	type stackTracer interface {
		StackTrace() []runtime.Frame
	}

	if st, ok := err.(stackTracer); ok {
		frames := st.StackTrace()
		if len(frames) > 0 {
			stackTrace := ""
			for _, frame := range frames {
				stackTrace += fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function)
			}
			return stackTrace
		}
	}

	// Check if error has a Stack() method (like github.com/pkg/errors)
	type stackError interface {
		Stack() []byte
	}

	if se, ok := err.(stackError); ok {
		stack := se.Stack()
		if len(stack) > 0 {
			return string(stack)
		}
	}

	// Check if error implements fmt.Formatter with +v verb for stack traces
	if formattable, ok := err.(fmt.Formatter); ok {
		stackTrace := fmt.Sprintf("%+v", formattable)
		// Only return if it looks like it contains stack trace information
		if len(stackTrace) > len(err.Error())+50 { // Heuristic: much longer than just the error message
			return stackTrace
		}
	}

	// If no stack trace is available, return empty string
	return ""
}
