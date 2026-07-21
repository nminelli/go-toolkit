package log_test

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/MFN-AISystems/go-toolkit/telemetry/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextWithLogger(t *testing.T) {
	logger := log.NewLogger()
	ctx := log.Context(context.Background(), logger)

	retrievedLogger := log.FromContext(ctx)
	require.NotNil(t, retrievedLogger)
	assert.Same(t, logger, retrievedLogger)
}

func TestContextWithGinContext(t *testing.T) {
	logger := log.NewLogger()
	ginCtx := &gin.Context{}
	ctx := log.Context(ginCtx, logger)

	retrievedLogger := log.FromContext(ctx)
	require.NotNil(t, retrievedLogger)
	assert.Same(t, logger, retrievedLogger)
}

func TestFromContextWithoutLogger(t *testing.T) {
	ctx := context.Background()
	logger := log.FromContext(ctx)
	assert.NotNil(t, logger)
}

func TestDebug(t *testing.T) {
	logger := log.NewLogger()
	ctx := log.Context(context.Background(), logger)

	assert.NotPanics(t, func() {
		log.Debug(ctx, "test message")
	})
}

func TestError(t *testing.T) {
	logger := log.NewLogger()
	ctx := log.Context(context.Background(), logger)

	assert.NotPanics(t, func() {
		log.Error(ctx, "test message")
	})
}

func TestInfo(t *testing.T) {
	logger := log.NewLogger()
	ctx := log.Context(context.Background(), logger)

	assert.NotPanics(t, func() {
		log.Info(ctx, "test message")
	})
}

func TestPanic(t *testing.T) {
	logger := log.NewLogger()
	ctx := log.Context(context.Background(), logger)

	assert.Panics(t, func() {
		log.Panic(ctx, "test message")
	})
}

func TestWarn(t *testing.T) {
	logger := log.NewLogger()
	ctx := log.Context(context.Background(), logger)

	assert.NotPanics(t, func() {
		log.Warn(ctx, "test message")
	})
}
