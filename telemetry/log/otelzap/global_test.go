package otelzap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestL(t *testing.T) {
	// Setup
	originalLogger := zap.L()
	testLogger, _ := zap.NewProduction()
	zap.ReplaceGlobals(testLogger)

	// Test
	logger := L()
	assert.NotNil(t, logger)
	assert.IsType(t, &Logger{}, logger)
	assert.Equal(t, testLogger, logger.Logger)

	// Cleanup
	zap.ReplaceGlobals(originalLogger)
}

func TestS(t *testing.T) {
	// Setup
	originalLogger := zap.L()
	testLogger, _ := zap.NewProduction()
	zap.ReplaceGlobals(testLogger)

	// Test
	sugared := S()
	assert.NotNil(t, sugared)
	assert.IsType(t, &SugaredLogger{}, sugared)
	assert.Equal(t, testLogger.Sugar(), sugared.SugaredLogger)

	// Cleanup
	zap.ReplaceGlobals(originalLogger)
}

func TestCtx(t *testing.T) {
	// Setup
	originalLogger := zap.L()
	testLogger, _ := zap.NewProduction()
	zap.ReplaceGlobals(testLogger)

	// Test cases
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{
			name: "With background context",
			ctx:  context.Background(),
		},
		{
			name: "With TODO context",
			ctx:  context.TODO(),
		},
		{
			name: "With value context",
			ctx:  context.WithValue(context.Background(), "key", "value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := Ctx(tt.ctx)
			assert.NotNil(t, logger)
			assert.IsType(t, &Logger{}, logger)
			assert.Equal(t, testLogger, logger.Logger)
		})
	}

	// Cleanup
	zap.ReplaceGlobals(originalLogger)
}

func TestGlobalIntegration(t *testing.T) {
	// Setup
	originalLogger := zap.L()
	testLogger, _ := zap.NewProduction()
	zap.ReplaceGlobals(testLogger)

	// Test chaining methods
	ctx := context.Background()

	// L() -> Ctx()
	l1 := L().Ctx(ctx)
	assert.NotNil(t, l1)
	assert.IsType(t, &Logger{}, l1)

	// Ctx() -> Sugar()
	s1 := Ctx(ctx).Sugar()
	assert.NotNil(t, s1)
	assert.IsType(t, &SugaredLogger{}, s1)

	// L() -> Sugar()
	s2 := L().Sugar()
	assert.NotNil(t, s2)
	assert.IsType(t, &SugaredLogger{}, s2)

	// Cleanup
	zap.ReplaceGlobals(originalLogger)
}
