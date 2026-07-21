package otelzap

import (
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	otel "github.com/agoda-com/opentelemetry-logs-go/logs"
	sdk "github.com/agoda-com/opentelemetry-logs-go/sdk/logs"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLevelEnabler_ChangeLevelAfterCreation(t *testing.T) {
	atomicLevel := zap.NewAtomicLevelAt(zap.WarnLevel)
	loggerProvider := sdk.NewLoggerProvider()
	core := NewOtelCore(loggerProvider, WithLevelEnabler(atomicLevel))

	assert.False(t, core.Enabled(zap.InfoLevel))

	atomicLevel.SetLevel(zap.InfoLevel)

	assert.True(t, core.Enabled(zap.InfoLevel))
}

type mockLogger struct {
	records []otel.LogRecord
}

func (m *mockLogger) Emit(record otel.LogRecord) {
	m.records = append(m.records, record)
}

func TestOtlpCore_Enabled(t *testing.T) {
	tests := []struct {
		name     string
		level    zapcore.Level
		expected bool
	}{
		{"Debug enabled", zapcore.DebugLevel, true},
		{"Info enabled", zapcore.InfoLevel, true},
		{"Warn enabled", zapcore.WarnLevel, true},
		{"Error enabled", zapcore.ErrorLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := NewOtelCore(sdk.NewLoggerProvider(), WithLevel(tt.level))
			assert.Equal(t, tt.expected, core.Enabled(tt.level))
		})
	}
}

func TestOtlpCore_With(t *testing.T) {
	logger := &mockLogger{}
	core := &otlpCore{
		logger:       logger,
		levelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true }),
	}

	fields := []zapcore.Field{
		zap.String("key1", "value1"),
		zap.Int("key2", 123),
	}

	newCore := core.With(fields)
	otlpCore, ok := newCore.(*otlpCore)

	assert.True(t, ok)
	assert.Equal(t, len(fields), len(otlpCore.fields))
	assert.Equal(t, fields[0].String, otlpCore.fields[0].String)
	assert.Equal(t, fields[1].Integer, otlpCore.fields[1].Integer)
}

func TestOtlpCore_Check(t *testing.T) {
	tests := []struct {
		name     string
		level    zapcore.Level
		enabled  bool
		expected bool
	}{
		{"Enabled level", zapcore.InfoLevel, true, true},
		{"Disabled level", zapcore.DebugLevel, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core := &otlpCore{
				logger: &mockLogger{},
				levelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool {
					return tt.enabled
				}),
			}

			ent := zapcore.Entry{Level: tt.level}
			checked := &zapcore.CheckedEntry{}
			result := core.Check(ent, checked)

			if tt.expected {
				assert.NotNil(t, result)
			} else {
				assert.Equal(t, checked, result)
			}
		})
	}
}

func TestOtlpCore_Write(t *testing.T) {
	logger := &mockLogger{}
	core := &otlpCore{
		logger:       logger,
		levelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true }),
	}
	core = core.With([]zapcore.Field{zap.String("deployment.environment", "local")}).(*otlpCore)

	tests := []struct {
		name          string
		entry         zapcore.Entry
		fields        []zapcore.Field
		expectedAttrs int
	}{
		{
			name: "Basic log entry",
			entry: zapcore.Entry{
				Level:   zapcore.InfoLevel,
				Time:    time.Now(),
				Message: "test message",
			},
			fields: []zapcore.Field{
				zap.String("key", "value"),
			},
			expectedAttrs: 3, // level + custom field
		},
		{
			name: "Error entry with stack trace but no error field",
			entry: zapcore.Entry{
				Level:   zapcore.ErrorLevel,
				Time:    time.Now(),
				Message: "error message",
				Stack:   "stack trace",
				Caller:  zapcore.EntryCaller{Defined: true, File: "file.go", Line: 123},
			},
			fields:        []zapcore.Field{},
			expectedAttrs: 2, // deployment.environment + level (no exception info without error field)
		},
		{
			name: "Error entry with error field",
			entry: zapcore.Entry{
				Level:   zapcore.ErrorLevel,
				Time:    time.Now(),
				Message: "error message",
			},
			fields: []zapcore.Field{
				zap.Error(errors.New("test error")),
			},
			expectedAttrs: 4, // deployment.environment + level + exception.type + exception.message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger.records = nil
			err := core.Write(tt.entry, tt.fields)

			assert.NoError(t, err)
			assert.Len(t, logger.records, 1)

			record := logger.records[0]
			attrs := *record.Attributes()
			assert.Len(t, attrs, tt.expectedAttrs)
		})
	}
}

func TestOtlpCore_TraceContext(t *testing.T) {
	logger := &mockLogger{}
	traceID := trace.TraceID{0x01}
	spanID := trace.SpanID{0x02}
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	core := &otlpCore{
		logger: logger,
		fields: []zapcore.Field{
			zap.Any("context", spanContext),
		},
		levelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true }),
	}

	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Time:    time.Now(),
		Message: "test with trace",
	}

	err := core.Write(entry, nil)
	assert.NoError(t, err)
	assert.Len(t, logger.records, 1)

	record := logger.records[0]
	assert.Equal(t, &traceID, record.TraceId())
	assert.Equal(t, &spanID, record.SpanId())
}

func TestOtlpCore_Sync(t *testing.T) {
	core := &otlpCore{
		logger: &mockLogger{},
	}
	assert.NoError(t, core.Sync())
}

func TestNewOtelCore(t *testing.T) {
	provider := sdk.NewLoggerProvider()
	core := NewOtelCore(provider)

	assert.NotNil(t, core)
	otlpCore, ok := core.(*otlpCore)
	assert.True(t, ok)
	assert.NotNil(t, otlpCore.logger)
	assert.NotNil(t, otlpCore.levelEnabler)
}

func TestWithLevelEnabler(t *testing.T) {
	enabler := zap.LevelEnablerFunc(func(l zapcore.Level) bool { return l >= zapcore.WarnLevel })
	provider := sdk.NewLoggerProvider()
	core := NewOtelCore(provider, WithLevelEnabler(enabler))

	otlpCore, ok := core.(*otlpCore)
	assert.True(t, ok)
	assert.False(t, otlpCore.Enabled(zapcore.InfoLevel))
	assert.True(t, otlpCore.Enabled(zapcore.WarnLevel))
}

// Custom error types for testing
type customError struct {
	message string
}

func (e *customError) Error() string {
	return e.message
}

type stackError struct {
	message string
	stack   []byte
}

func (e *stackError) Error() string {
	return e.message
}

func (e *stackError) Stack() []byte {
	return e.stack
}

type stackTracerError struct {
	message string
	frames  []runtime.Frame
}

func (e *stackTracerError) Error() string {
	return e.message
}

func (e *stackTracerError) StackTrace() []runtime.Frame {
	return e.frames
}

type formatterError struct {
	message   string
	stackInfo string
}

func (e *formatterError) Error() string {
	return e.message
}

func (e *formatterError) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('+') {
		fmt.Fprintf(f, "%s\n%s", e.message, e.stackInfo)
	} else {
		fmt.Fprintf(f, "%s", e.message)
	}
}

func TestOtlpCore_GetErrorType(t *testing.T) {
	core := &otlpCore{}

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "standard error",
			err:      errors.New("test error"),
			expected: "errors.errorString",
		},
		{
			name:     "custom error",
			err:      &customError{message: "custom test error"},
			expected: "github.com/MFN-AISystems/go-toolkit/telemetry/log/otelzap.customError",
		},
		{
			name:     "fmt error",
			err:      fmt.Errorf("formatted error: %s", "test"),
			expected: "errors.errorString",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.getErrorType(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOtlpCore_GetErrorStackTrace(t *testing.T) {
	core := &otlpCore{}

	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "standard error without stack",
			err:      errors.New("test error"),
			expected: "",
		},
		{
			name:     "error with Stack() method",
			err:      &stackError{message: "stack error", stack: []byte("stack trace content")},
			expected: "stack trace content",
		},
		{
			name: "error with StackTrace() method",
			err: &stackTracerError{
				message: "stack tracer error",
				frames: []runtime.Frame{
					{File: "file1.go", Line: 10, Function: "func1"},
					{File: "file2.go", Line: 20, Function: "func2"},
				},
			},
			expected: "file1.go:10 func1\nfile2.go:20 func2\n",
		},
		{
			name:     "error with formatter (short)",
			err:      &formatterError{message: "short error", stackInfo: "short"},
			expected: "",
		},
		{
			name: "error with formatter (long stack)",
			err: &formatterError{
				message:   "formatter error",
				stackInfo: "this is a very long stack trace that should be much longer than the error message to trigger the heuristic check for stack trace detection",
			},
			expected: "formatter error\nthis is a very long stack trace that should be much longer than the error message to trigger the heuristic check for stack trace detection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := core.getErrorStackTrace(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOtlpCore_AddExceptionInfo(t *testing.T) {
	core := &otlpCore{}

	tests := []struct {
		name               string
		fields             []zapcore.Field
		expectedException  bool
		expectedType       string
		expectedMessage    string
		expectedStackTrace string
	}{
		{
			name:              "no error fields",
			fields:            []zapcore.Field{zap.String("key", "value")},
			expectedException: false,
		},
		{
			name:               "error field with standard error",
			fields:             []zapcore.Field{zap.Error(errors.New("test error"))},
			expectedException:  true,
			expectedType:       "errors.errorString",
			expectedMessage:    "test error",
			expectedStackTrace: "",
		},
		{
			name:               "error field with custom error",
			fields:             []zapcore.Field{zap.Error(&customError{message: "custom error"})},
			expectedException:  true,
			expectedType:       "github.com/MFN-AISystems/go-toolkit/telemetry/log/otelzap.customError",
			expectedMessage:    "custom error",
			expectedStackTrace: "",
		},
		{
			name: "error field with stack trace",
			fields: []zapcore.Field{
				zap.Error(&stackError{message: "stack error", stack: []byte("test stack")}),
			},
			expectedException:  true,
			expectedType:       "github.com/MFN-AISystems/go-toolkit/telemetry/log/otelzap.stackError",
			expectedMessage:    "stack error",
			expectedStackTrace: "test stack",
		},
		{
			name: "multiple fields with error",
			fields: []zapcore.Field{
				zap.String("key", "value"),
				zap.Error(errors.New("test error")),
				zap.Int("count", 42),
			},
			expectedException:  true,
			expectedType:       "errors.errorString",
			expectedMessage:    "test error",
			expectedStackTrace: "",
		},
		{
			name: "multiple error fields (only first processed)",
			fields: []zapcore.Field{
				zap.Error(errors.New("first error")),
				zap.Error(errors.New("second error")),
			},
			expectedException:  true,
			expectedType:       "errors.errorString",
			expectedMessage:    "first error",
			expectedStackTrace: "",
		},
		{
			name:              "error field with nil error",
			fields:            []zapcore.Field{zap.NamedError("error", nil)},
			expectedException: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attributes := []attribute.KeyValue{}

			result := core.addExceptionInfo(attributes, tt.fields)

			if !tt.expectedException {
				assert.Equal(t, len(attributes), len(result))
				return
			}

			// Find exception attributes
			var exceptionType, exceptionMessage, exceptionStackTrace string
			for _, attr := range result {
				switch attr.Key {
				case semconv.ExceptionTypeKey:
					exceptionType = attr.Value.AsString()
				case "exception.message":
					exceptionMessage = attr.Value.AsString()
				case semconv.ExceptionStacktraceKey:
					exceptionStackTrace = attr.Value.AsString()
				}
			}

			assert.Equal(t, tt.expectedType, exceptionType)
			assert.Equal(t, tt.expectedMessage, exceptionMessage)
			if tt.expectedStackTrace != "" {
				assert.Equal(t, tt.expectedStackTrace, exceptionStackTrace)
			}
		})
	}
}

func TestOtlpCore_AddExceptionInfo_Integration(t *testing.T) {
	logger := &mockLogger{}
	core := &otlpCore{
		logger:       logger,
		levelEnabler: zap.LevelEnablerFunc(func(l zapcore.Level) bool { return true }),
	}

	entry := zapcore.Entry{
		Level:   zapcore.ErrorLevel,
		Time:    time.Now(),
		Message: "test error occurred",
	}

	fields := []zapcore.Field{
		zap.String("context", "test context"),
		zap.Error(errors.New("integration test error")),
		zap.Int("code", 500),
	}

	err := core.Write(entry, fields)
	assert.NoError(t, err)
	assert.Len(t, logger.records, 1)

	record := logger.records[0]
	attrs := *record.Attributes()

	// Verify that exception attributes are present
	var hasExceptionType, hasExceptionMessage bool
	for _, attr := range attrs {
		if attr.Key == semconv.ExceptionTypeKey {
			hasExceptionType = true
			assert.Equal(t, "errors.errorString", attr.Value.AsString())
		}
		if attr.Key == "exception.message" {
			hasExceptionMessage = true
			assert.Equal(t, "integration test error", attr.Value.AsString())
		}
	}

	assert.True(t, hasExceptionType, "Expected exception.type attribute")
	assert.True(t, hasExceptionMessage, "Expected exception.message attribute")
}
