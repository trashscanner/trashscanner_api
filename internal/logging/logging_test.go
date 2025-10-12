package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.Config
		expectError bool
	}{
		{
			name: "json format with debug level",
			cfg: config.Config{
				Log: config.LogConfig{
					Level:  "debug",
					Format: "json",
				},
			},
			expectError: false,
		},
		{
			name: "text format with info level",
			cfg: config.Config{
				Log: config.LogConfig{
					Level:  "info",
					Format: "text",
				},
			},
			expectError: false,
		},
		{
			name: "invalid level defaults to info",
			cfg: config.Config{
				Log: config.LogConfig{
					Level:  "invalid",
					Format: "json",
				},
			},
			expectError: false,
		},
		{
			name: "default format (empty)",
			cfg: config.Config{
				Log: config.LogConfig{
					Level:  "warn",
					Format: "",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.cfg)
			if tt.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, logger)
			assert.NotNil(t, logger.Entry)
		})
	}
}

func TestNewLogger_WithFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
			File:   logFile,
		},
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("test message")

	_, err = os.Stat(logFile)
	assert.NoError(t, err, "log file should be created")
}

func TestNewLogger_WithInvalidFile(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
			File:   "/invalid/path/that/does/not/exist/test.log",
		},
	}

	_, err := NewLogger(cfg)
	assert.Error(t, err)
}

func TestLogger_WithComponent(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	tests := []struct {
		name      string
		component Component
	}{
		{"main component", MainComponent},
		{"api component", ApiComponent},
		{"custom component", Component("CUSTOM")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := logger.WithComponent(tt.component)
			assert.NotNil(t, l)
			assert.Equal(t, tt.component, l.Data["component"])
		})
	}
}

func TestLogger_WithTags(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	t.Run("api tag", func(t *testing.T) {
		l := logger.WithApiTag()
		assert.NotNil(t, l)
		assert.Equal(t, ApiComponent, l.Data["component"])
	})
}

func TestLogger_WithField(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	l := logger.WithField("test_key", "test_value")
	assert.NotNil(t, l)
	assert.Equal(t, "test_value", l.Data["test_key"])
}

func TestLogger_WithContext_User(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	userID := uuid.New()
	user := models.User{
		ID:    userID,
		Login: "testuser",
	}

	ctx := utils.SetUser(context.Background(), user)
	l := logger.WithContext(ctx)

	assert.NotNil(t, l)
	assert.Equal(t, userID.String(), l.Data["user_id"])
	assert.Equal(t, "testuser", l.Data["user_login"])
}

func TestLogger_WithContext_RequestID(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), utils.RequestIDKey, "test-request-id")
	l := logger.WithContext(ctx)

	assert.NotNil(t, l)
	assert.Equal(t, "test-request-id", l.Data["request_id"])
}

func TestLogger_WithContext_PathAndMethod(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.PathKey, "/api/v1/users")
	ctx = context.WithValue(ctx, utils.MethodKey, "GET")

	l := logger.WithContext(ctx)

	assert.NotNil(t, l)
	assert.Equal(t, "/api/v1/users", l.Data["path"])
	assert.Equal(t, "GET", l.Data["method"])
}

func TestLogger_WithContext_AllFields(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	userID := uuid.New()
	user := models.User{
		ID:    userID,
		Login: "testuser",
	}

	ctx := context.Background()
	ctx = utils.SetUser(ctx, user)
	ctx = context.WithValue(ctx, utils.RequestIDKey, "test-request-id")
	ctx = context.WithValue(ctx, utils.PathKey, "/api/v1/users")
	ctx = context.WithValue(ctx, utils.MethodKey, "POST")

	l := logger.WithContext(ctx)

	assert.NotNil(t, l)
	assert.Equal(t, userID.String(), l.Data["user_id"])
	assert.Equal(t, "testuser", l.Data["user_login"])
	assert.Equal(t, "test-request-id", l.Data["request_id"])
	assert.Equal(t, "/api/v1/users", l.Data["path"])
	assert.Equal(t, "POST", l.Data["method"])
}

func TestLogger_WithContext_EmptyContext(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	l := logger.WithContext(ctx)

	assert.NotNil(t, l)
	// Should return same logger if no fields
	assert.Equal(t, logger, l)
}

func TestLogger_WithContext_IgnoresTimeAndRequestBody(t *testing.T) {
	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	ctx = context.WithValue(ctx, utils.TimeKey, "some-time")
	ctx = context.WithValue(ctx, utils.RequestBodyKey, "some-body")

	l := logger.WithContext(ctx)

	assert.NotNil(t, l)
	_, hasTime := l.Data["time"]
	_, hasBody := l.Data["request_body"]
	assert.False(t, hasTime, "time should not be in log fields")
	assert.False(t, hasBody, "request_body should not be in log fields")
}

func TestLogger_LogOutput(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)

	logger.Logger.SetOutput(&buf)

	logger.Info("test message")

	var logEntry map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "test message", logEntry["message"])
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "MAIN", logEntry["component"])
}

func TestLogger_DifferentLogLevels(t *testing.T) {
	var buf bytes.Buffer

	cfg := config.Config{
		Log: config.LogConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	logger, err := NewLogger(cfg)
	require.NoError(t, err)
	logger.Logger.SetOutput(&buf)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}
