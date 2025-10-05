package utils

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func TestSetAndGetUser(t *testing.T) {
	ctx := context.Background()
	user := models.User{ID: uuid.New(), Login: "tester"}

	ctxWithUser := SetUser(ctx, user)
	assert.Nil(t, GetUser(ctx))

	retrieved, ok := GetUser(ctxWithUser).(models.User)
	require.True(t, ok)
	assert.Equal(t, user, retrieved)
}

func TestSetAndGetRequestBody(t *testing.T) {
	ctx := context.Background()
	body := map[string]string{"key": "value"}

	ctxWithBody := SetRequestBody(ctx, body)

	retrieved := GetRequestBody(ctxWithBody)
	require.NotNil(t, retrieved)
	assert.Equal(t, body, retrieved)
}

func TestSetAndGetRequestID(t *testing.T) {
	ctx := context.Background()
	ctxWithID := SetRequestID(ctx, "req-123")

	id, ok := GetRequestID(ctxWithID)
	require.True(t, ok)
	assert.Equal(t, "req-123", id)
}

func TestGetRequestIDMissing(t *testing.T) {
	id, ok := GetRequestID(context.Background())
	assert.False(t, ok)
	assert.Empty(t, id)
}
