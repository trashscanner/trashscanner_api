package utils

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func TestSetAndGetUser(t *testing.T) {
	ctx := context.Background()
	user := models.User{ID: uuid.New(), Login: "tester"}

	ctxWithUser := SetUser(ctx, &user)
	assert.Equal(t, &user, GetUser(ctxWithUser))
}

func TestSetAndGetRequestBody(t *testing.T) {
	ctx := context.Background()
	body := map[string]string{"key": "value"}

	ctxWithBody := SetRequestBody(ctx, body)

	retrieved := GetRequestBody(ctxWithBody)
	require.NotNil(t, retrieved)
	assert.Equal(t, body, retrieved)
}

func TestGetRequestIDMissing(t *testing.T) {
	id, ok := GetRequestID(context.Background())
	assert.False(t, ok)
	assert.Empty(t, id)
}

func TestGetPath(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, PathKey, "/api/users")

	path, ok := GetPath(ctx)
	require.True(t, ok)
	assert.Equal(t, "/api/users", path)
}

func TestGetPathMissing(t *testing.T) {
	path, ok := GetPath(context.Background())
	assert.False(t, ok)
	assert.Empty(t, path)
}

func TestGetMethod(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, MethodKey, "POST")

	method, ok := GetMethod(ctx)
	require.True(t, ok)
	assert.Equal(t, "POST", method)
}

func TestGetMethodMissing(t *testing.T) {
	method, ok := GetMethod(context.Background())
	assert.False(t, ok)
	assert.Empty(t, method)
}

func TestGetContextValue(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, RequestIDKey, "test-id")

	val, ok := GetContextValue(ctx, RequestIDKey)
	require.True(t, ok)
	assert.Equal(t, "test-id", val)
}

func TestGetContextValueMissing(t *testing.T) {
	val, ok := GetContextValue(context.Background(), RequestIDKey)
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestElapsedTime(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, TimeKey, time.Now().Add(-100*time.Millisecond))

	elapsed, ok := ElapsedTime(ctx)
	require.True(t, ok)
	assert.GreaterOrEqual(t, elapsed, time.Duration(100))
}

func TestElapsedTimeMissing(t *testing.T) {
	elapsed, ok := ElapsedTime(context.Background())
	assert.False(t, ok)
	assert.Equal(t, time.Duration(0), elapsed)
}

func TestContextKeys(t *testing.T) {
	// Verify all expected keys are present in the map
	expectedKeys := []ContextKey{
		UserCtxKey,
		RequestBodyKey,
		RequestIDKey,
		TimeKey,
		PathKey,
		MethodKey,
	}

	for _, key := range expectedKeys {
		_, exists := ContextKeys[key]
		assert.True(t, exists, "Expected key %s to be in ContextKeys", key)
	}

	assert.Equal(t, len(expectedKeys), len(ContextKeys))
}

func TestCopyContext(t *testing.T) {
	t.Run("Copies all known values", func(t *testing.T) {
		ctx := context.Background()
		user := &models.User{ID: uuid.New(), Login: "testuser"}
		ctx = SetUser(ctx, user)
		ctx = context.WithValue(ctx, RequestIDKey, "req-123")
		ctx = context.WithValue(ctx, PathKey, "/api/test")
		ctx = context.WithValue(ctx, MethodKey, "POST")
		ctx = context.WithValue(ctx, TimeKey, time.Now())

		newCtx := CopyContext(ctx)

		assert.Equal(t, user, GetUser(newCtx))

		reqID, ok := GetRequestID(newCtx)
		assert.True(t, ok)
		assert.Equal(t, "req-123", reqID)

		path, ok := GetPath(newCtx)
		assert.True(t, ok)
		assert.Equal(t, "/api/test", path)

		method, ok := GetMethod(newCtx)
		assert.True(t, ok)
		assert.Equal(t, "POST", method)

		_, ok = ElapsedTime(newCtx)
		assert.True(t, ok)
	})

	t.Run("New context is independent from original", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ctx = context.WithValue(ctx, RequestIDKey, "req-456")

		newCtx := CopyContext(ctx)

		cancel()

		select {
		case <-ctx.Done():
		default:
			t.Fatal("Original context should be cancelled")
		}

		select {
		case <-newCtx.Done():
			t.Fatal("New context should NOT be cancelled")
		default:
		}

		reqID, ok := GetRequestID(newCtx)
		assert.True(t, ok)
		assert.Equal(t, "req-456", reqID)
	})

	t.Run("Handles empty context", func(t *testing.T) {
		ctx := context.Background()

		newCtx := CopyContext(ctx)

		_, ok := GetRequestID(newCtx)
		assert.False(t, ok)

		_, ok = GetPath(newCtx)
		assert.False(t, ok)
	})

	t.Run("Skips nil values", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, RequestIDKey, "req-789")

		newCtx := CopyContext(ctx)

		reqID, ok := GetRequestID(newCtx)
		assert.True(t, ok)
		assert.Equal(t, "req-789", reqID)

		val, ok := GetContextValue(newCtx, UserCtxKey)
		assert.False(t, ok)
		assert.Nil(t, val)
	})
}
