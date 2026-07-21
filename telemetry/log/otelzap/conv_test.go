package otelzap

import (
	"bytes"
	"errors"
	"math"
	"testing"
	"time"

	otel "github.com/agoda-com/opentelemetry-logs-go/logs"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	testFieldKey = "test-123"
	testNow      = time.Now()
)

func TestOtelLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    zapcore.Level
		expected otel.SeverityNumber
	}{
		{"Debug", zapcore.DebugLevel, otel.DEBUG},
		{"Info", zapcore.InfoLevel, otel.INFO},
		{"Warn", zapcore.WarnLevel, otel.WARN},
		{"Error", zapcore.ErrorLevel, otel.ERROR},
		{"DPanic", zapcore.DPanicLevel, otel.ERROR},
		{"Panic", zapcore.PanicLevel, otel.ERROR},
		{"Fatal", zapcore.FatalLevel, otel.FATAL},
		{"Unknown", zapcore.Level(99), otel.TRACE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := otelLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type _testType struct{}

func TestOTeLAttributeMapping(t *testing.T) {
	var nilPtr *_testType

	tests := []struct {
		Name     string
		Input    zapcore.Field
		Expected []attribute.KeyValue
	}{
		{Name: "UnknownType", Input: zapcore.Field{Key: testFieldKey, Type: zapcore.UnknownType, String: "hello"}, Expected: []attribute.KeyValue{attribute.String(testFieldKey, "hello")}},
		{Name: "Bool", Input: zap.Bool(testFieldKey, true), Expected: []attribute.KeyValue{attribute.Bool(testFieldKey, true)}},
		{Name: "Float64", Input: zap.Float64(testFieldKey, 123.123), Expected: []attribute.KeyValue{attribute.Float64(testFieldKey, 123.123)}},
		{Name: "Float32", Input: zap.Float32(testFieldKey, 123.123), Expected: []attribute.KeyValue{attribute.Float64(testFieldKey, float64(float32(123.123)))}},
		{Name: "Int", Input: zap.Int(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "Int32", Input: zap.Int32(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "Int16", Input: zap.Int16(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "Int8", Input: zap.Int8(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "String", Input: zap.String(testFieldKey, "hello"), Expected: []attribute.KeyValue{attribute.String(testFieldKey, "hello")}},
		{Name: "Uint", Input: zap.Uint(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "Uint32", Input: zap.Uint32(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "Uint16", Input: zap.Uint16(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "Uint8", Input: zap.Uint8(testFieldKey, 123), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, 123)}},
		{Name: "ByteString", Input: zap.ByteString(testFieldKey, []byte("hello")), Expected: []attribute.KeyValue{attribute.String(testFieldKey, "hello")}},
		{Name: "Binary", Input: zap.Binary(testFieldKey, []byte{1, 0, 0, 1}), Expected: []attribute.KeyValue{attribute.String(testFieldKey, "AQAAAQ==")}},
		{Name: "Duration", Input: zap.Duration(testFieldKey, time.Minute), Expected: []attribute.KeyValue{attribute.Float64(testFieldKey, time.Minute.Seconds())}},
		{Name: "Time", Input: zap.Time(testFieldKey, testNow), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, testNow.Unix())}},
		{Name: "Time", Input: zap.Time(testFieldKey, testNow.In(time.UTC)), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, testNow.Unix())}},
		{Name: "TimeType_Invalid", Input: zap.Field{Key: testFieldKey, Type: zapcore.TimeType, Interface: nil}, Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, time.Unix(0, int64(0)).Unix())}},
		{Name: "TimeFullType", Input: zap.Time(testFieldKey, time.Unix(0, math.MaxInt64).Add(1*time.Second)), Expected: []attribute.KeyValue{attribute.Int64(testFieldKey, time.Unix(0, math.MaxInt64).Add(1*time.Second).Unix())}},
		{Name: "Stringer", Input: zap.Stringer(testFieldKey, bytes.NewBuffer([]byte("hello"))), Expected: []attribute.KeyValue{attribute.String(testFieldKey, "hello")}},
		{Name: "Stringer_Ptr", Input: zap.Field{Key: testFieldKey, Type: zapcore.StringerType, Interface: nilPtr}, Expected: []attribute.KeyValue{attribute.String(testFieldKey, "<nil>")}},
		{Name: "Stringer_Default", Input: zap.Field{Key: testFieldKey, Type: zapcore.StringerType}, Expected: []attribute.KeyValue{}},
		{Name: "Error", Input: zap.Error(errors.New("world")), Expected: []attribute.KeyValue{semconv.ExceptionMessage("world")}},
		{Name: "ErrorType", Input: zapcore.Field{Key: testFieldKey, Type: zapcore.ErrorType, String: "asd"}, Expected: []attribute.KeyValue{}},
		{Name: "Skip", Input: zap.Skip(), Expected: []attribute.KeyValue{}},
		{Name: "Default", Input: zapcore.Field{Key: testFieldKey, Type: 99, String: "something"}, Expected: []attribute.KeyValue{attribute.String(testFieldKey, "something")}},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			output := otelAttribute(test.Input)
			assert.ElementsMatch(t, test.Expected, output)
		})
	}
}

// TestStruct for testing ReflectType handling
type TestStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestOtelAttribute_ReflectType(t *testing.T) {
	tests := []struct {
		name        string
		field       zapcore.Field
		expectedKey string
		expectedVal string
	}{
		{
			name:        "Basic struct (ReflectType)",
			field:       zap.Any("test_struct", TestStruct{Name: "John", Age: 30}),
			expectedKey: "test_struct",
			expectedVal: `{"name":"John","age":30}`,
		},
		{
			name:        "Nil interface (ReflectType)",
			field:       zap.Any("test_nil", nil),
			expectedKey: "test_nil",
			expectedVal: "<nil>",
		},
		{
			name:        "Simple map (ReflectType)",
			field:       zap.Any("test_map", map[string]interface{}{"key": "value", "number": 42}),
			expectedKey: "test_map",
			// Note: map iteration order is not guaranteed, so we'll check this differently
		},
		{
			name:        "Slice (ArrayMarshalerType)",
			field:       zap.Any("test_slice", []string{"a", "b", "c"}),
			expectedKey: "test_slice",
			expectedVal: `["a","b","c"]`,
		},
		{
			name:        "Integer slice (ArrayMarshalerType)",
			field:       zap.Any("test_int_slice", []int{1, 2, 3}),
			expectedKey: "test_int_slice",
			expectedVal: `[1,2,3]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := otelAttribute(tt.field)

			assert.Len(t, attrs, 1)
			assert.Equal(t, tt.expectedKey, string(attrs[0].Key))

			if tt.name == "Simple map (ReflectType)" {
				// For maps, just check that it's valid JSON and contains expected keys
				val := attrs[0].Value.AsString()
				assert.Contains(t, val, `"key":"value"`)
				assert.Contains(t, val, `"number":42`)
			} else {
				assert.Equal(t, tt.expectedVal, attrs[0].Value.AsString())
			}
			assert.Equal(t, attribute.STRING, attrs[0].Value.Type())
		})
	}
}

func TestOtelAttribute_ReflectType_JSONMarshalError(t *testing.T) {
	// Test with a type that can't be JSON marshaled (like a channel)
	ch := make(chan string)
	field := zap.Any("test_channel", ch)

	attrs := otelAttribute(field)

	assert.Len(t, attrs, 1)
	assert.Equal(t, "test_channel", string(attrs[0].Key))
	// Should fallback to fmt.Sprintf("%+v", value) when JSON marshal fails
	val := attrs[0].Value.AsString()
	// Channel values in Go show as memory addresses in %+v format
	assert.Contains(t, val, "0x")
}
