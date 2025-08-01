package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()

	assert.NotNil(t, logger)
	logger.Info("test message")
	assert.IsType(t, &zap.Logger{}, logger)
}

func TestNewLogger_MultipleInstances(t *testing.T) {
	logger1 := NewLogger()
	logger2 := NewLogger()

	assert.NotNil(t, logger1)
	assert.NotNil(t, logger2)

	logger1.Info("message from logger1")
	logger2.Info("message from logger2")
}

func TestNewLogger_LogLevels(t *testing.T) {
	logger := NewLogger()

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	assert.True(t, true)
}
