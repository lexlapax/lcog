package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected Config
	}{
		{
			name:   "Default config",
			config: DefaultConfig(),
			expected: Config{
				Level:  InfoLevel,
				Format: TextFormat,
			},
		},
		{
			name: "Custom config",
			config: Config{
				Level:  DebugLevel,
				Format: JSONFormat,
			},
			expected: Config{
				Level:  DebugLevel,
				Format: JSONFormat,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestLoggerSetup(t *testing.T) {
	var buf bytes.Buffer

	// Test text format
	textCfg := Config{
		Level:  InfoLevel,
		Format: TextFormat,
	}
	logger := SetupWithOutput(textCfg, &buf)
	require.NotNil(t, logger)

	logger.Info("test message", "key", "value")
	output := buf.String()
	
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "key=value")
	
	// Test JSON format
	buf.Reset()
	jsonCfg := Config{
		Level:  InfoLevel,
		Format: JSONFormat,
	}
	logger = SetupWithOutput(jsonCfg, &buf)
	require.NotNil(t, logger)

	logger.Info("test message", "key", "value")
	output = buf.String()
	
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(output), &logEntry)
	require.NoError(t, err)
	
	assert.Equal(t, "test message", logEntry["msg"])
	assert.Equal(t, "value", logEntry["key"])
}

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer

	// Test debug level
	cfg := Config{
		Level:  DebugLevel,
		Format: TextFormat,
	}
	logger := SetupWithOutput(cfg, &buf)
	require.NotNil(t, logger)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
	
	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
	
	// Test info level (debug should be filtered out)
	buf.Reset()
	cfg.Level = InfoLevel
	logger = SetupWithOutput(cfg, &buf)

	logger.Debug("debug message") // This should not appear
	logger.Info("info message")
	
	output = buf.String()
	assert.NotContains(t, output, "debug message")
	assert.Contains(t, output, "info message")
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	
	cfg := Config{
		Level:  DebugLevel,
		Format: TextFormat,
	}
	logger := SetupWithOutput(cfg, &buf)

	// Create a context with the logger
	ctx := WithLogger(context.Background(), logger)
	
	// Get logger from context
	loggerFromCtx := FromContext(ctx)
	loggerFromCtx.Info("context logger test")
	
	output := buf.String()
	assert.Contains(t, output, "context logger test")
	
	// Test with entity context
	buf.Reset()
	entityCtx := entity.NewContext("test-entity", "test-user")
	loggerWithEntity := WithEntityContext(logger, entityCtx)
	
	loggerWithEntity.Info("entity context test")
	
	output = buf.String()
	assert.Contains(t, output, "entity context test")
	assert.Contains(t, output, "entity_id=test-entity")
	assert.Contains(t, output, "user_id=test-user")
}

func TestHelperFunctions(t *testing.T) {
	// Test global helper functions
	t.Run("Global helper functions", func(t *testing.T) {
		var buf bytes.Buffer
		
		cfg := Config{
			Level:  DebugLevel,
			Format: TextFormat,
		}
		logger := SetupWithOutput(cfg, &buf)
		slog.SetDefault(logger)
		
		Debug("debug global")
		output := buf.String()
		assert.Contains(t, output, "debug global")
		
		buf.Reset()
		Info("info global")
		output = buf.String()
		assert.Contains(t, output, "info global")
		
		buf.Reset()
		Warn("warn global")
		output = buf.String()
		assert.Contains(t, output, "warn global")
		
		buf.Reset()
		Error("error global")
		output = buf.String()
		assert.Contains(t, output, "error global")
	})
	
	// Test context helper functions
	t.Run("Context helper functions", func(t *testing.T) {
		var buf bytes.Buffer
		
		cfg := Config{
			Level:  DebugLevel,
			Format: TextFormat,
		}
		logger := SetupWithOutput(cfg, &buf)
		
		// Create a context with the logger
		ctx := WithLogger(context.Background(), logger)
		
		DebugContext(ctx, "debug context")
		output := buf.String()
		assert.Contains(t, output, "debug context")
		
		buf.Reset()
		InfoContext(ctx, "info context")
		output = buf.String()
		assert.Contains(t, output, "info context")
		
		buf.Reset()
		WarnContext(ctx, "warn context")
		output = buf.String()
		assert.Contains(t, output, "warn context")
		
		buf.Reset()
		ErrorContext(ctx, "error context")
		output = buf.String()
		assert.Contains(t, output, "error context")
	})
}