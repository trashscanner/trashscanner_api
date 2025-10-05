package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store/mocks"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func setupTestManager(t *testing.T) (AuthManager, *mocks.Store, models.User) {
	utils.GenerateAndSetKeys()

	cfg := config.Config{
		Auth: config.AuthManagerConfig{
			Algorithm:       "EdDSA",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}

	generator, err := newJWTGenerator(cfg)
	require.NoError(t, err)

	mockStore := mocks.NewStore(t)

	manager := &jwtManager{
		generator: generator,
		store:     mockStore,
	}

	user := models.User{
		ID:    uuid.New(),
		Login: "testuser",
	}

	return manager, mockStore, user
}

func TestJWTManager_CreateNewPair(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)

		require.NoError(t, err)
		assert.NotEmpty(t, tokens.Access)
		assert.NotEmpty(t, tokens.Refresh)
		mockStore.AssertExpectations(t)
	})

	t.Run("revoke_error", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		expectedErr := errlocal.NewErrInternal("revoke error", "db error", nil)
		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(expectedErr).Once()

		tokens, err := manager.CreateNewPair(ctx, user)

		require.Error(t, err)
		assert.Nil(t, tokens)
		mockStore.AssertExpectations(t)
	})

	t.Run("store_error", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()

		expectedErr := errlocal.NewErrInternal("database error", "db error", nil)
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(expectedErr).Once()

		tokens, err := manager.CreateNewPair(ctx, user)

		require.Error(t, err)
		assert.Nil(t, tokens)
		mockStore.AssertExpectations(t)
	})

	t.Run("tokens_are_different", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Twice()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Twice()

		tokens1, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		tokens2, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		assert.NotEqual(t, tokens1.Access, tokens2.Access)
		assert.NotEqual(t, tokens1.Refresh, tokens2.Refresh)
		mockStore.AssertExpectations(t)
	})

	t.Run("revokes_old_tokens_before_creating_new", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		callOrder := 0
		revokeOrder := 0
		insertOrder := 0

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Run(func(args mock.Arguments) {
				callOrder++
				revokeOrder = callOrder
			}).
			Return(nil).Once()

		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Run(func(args mock.Arguments) {
				callOrder++
				insertOrder = callOrder
			}).
			Return(nil).Once()

		_, err := manager.CreateNewPair(ctx, user)

		require.NoError(t, err)
		assert.Less(t, revokeOrder, insertOrder, "RevokeAllUserTokens should be called before InsertRefreshToken")
		mockStore.AssertExpectations(t)
	})
}

func TestJWTManager_Refresh(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		oldTokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		storedToken := &models.RefreshToken{
			UserID:    user.ID,
			TokenHash: utils.HashToken(oldTokens.Refresh),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		}

		mockStore.On("GetUser", ctx, user.ID, false).
			Return(&user, nil).Once()
		mockStore.On("GetRefreshTokenByHash", ctx, utils.HashToken(oldTokens.Refresh)).
			Return(storedToken, nil).Once()
		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		newTokens, err := manager.Refresh(ctx, oldTokens.Refresh)

		require.NoError(t, err)
		assert.NotEmpty(t, newTokens.Access)
		assert.NotEmpty(t, newTokens.Refresh)
		assert.NotEqual(t, oldTokens.Access, newTokens.Access)
		assert.NotEqual(t, oldTokens.Refresh, newTokens.Refresh)
		mockStore.AssertExpectations(t)
	})

	t.Run("invalid_token_format", func(t *testing.T) {
		manager, mockStore, _ := setupTestManager(t)
		ctx := context.Background()

		_, err := manager.Refresh(ctx, "invalid.token")

		require.Error(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("token_not_found_in_db", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		mockStore.On("GetUser", ctx, user.ID, false).
			Return(&user, nil).Once()

		expectedErr := errlocal.NewErrNotFound("token not found", "", nil)
		mockStore.On("GetRefreshTokenByHash", ctx, utils.HashToken(tokens.Refresh)).
			Return(nil, expectedErr).Once()

		_, err = manager.Refresh(ctx, tokens.Refresh)

		require.Error(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("revoked_token", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		revokedToken := &models.RefreshToken{
			UserID:    user.ID,
			TokenHash: utils.HashToken(tokens.Refresh),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   true,
		}

		mockStore.On("GetUser", ctx, user.ID, false).
			Return(&user, nil).Once()
		mockStore.On("GetRefreshTokenByHash", ctx, utils.HashToken(tokens.Refresh)).
			Return(revokedToken, nil).Once()

		_, err = manager.Refresh(ctx, tokens.Refresh)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "revoked")
		mockStore.AssertExpectations(t)
	})

	t.Run("expired_token", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		expiredToken := &models.RefreshToken{
			UserID:    user.ID,
			TokenHash: utils.HashToken(tokens.Refresh),
			ExpiresAt: time.Now().Add(-24 * time.Hour),
			Revoked:   false,
		}

		mockStore.On("GetUser", ctx, user.ID, false).
			Return(&user, nil).Once()
		mockStore.On("GetRefreshTokenByHash", ctx, utils.HashToken(tokens.Refresh)).
			Return(expiredToken, nil).Once()

		_, err = manager.Refresh(ctx, tokens.Refresh)

		require.Error(t, err)
		assert.Equal(t, jwt.ErrTokenExpired, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("user_not_found", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		expectedErr := errlocal.NewErrNotFound("user not found", "", nil)
		mockStore.On("GetUser", ctx, user.ID, false).
			Return(nil, expectedErr).Once()

		_, err = manager.Refresh(ctx, tokens.Refresh)

		require.Error(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("create_new_pair_error", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		storedToken := &models.RefreshToken{
			UserID:    user.ID,
			TokenHash: utils.HashToken(tokens.Refresh),
			ExpiresAt: time.Now().Add(24 * time.Hour),
			Revoked:   false,
		}

		mockStore.On("GetUser", ctx, user.ID, false).
			Return(&user, nil).Once()
		mockStore.On("GetRefreshTokenByHash", ctx, utils.HashToken(tokens.Refresh)).
			Return(storedToken, nil).Once()

		expectedErr := errlocal.NewErrInternal("revoke error", "db error", nil)
		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(expectedErr).Once()

		_, err = manager.Refresh(ctx, tokens.Refresh)

		require.Error(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestJWTManager_Parse(t *testing.T) {
	t.Run("valid_access_token", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		claims, err := manager.Parse(tokens.Access)

		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), claims.UserID)
		assert.Equal(t, user.Login, claims.Login)
		assert.Equal(t, "access", claims.TokenType)
		mockStore.AssertExpectations(t)
	})

	t.Run("invalid_token", func(t *testing.T) {
		manager, mockStore, _ := setupTestManager(t)

		_, err := manager.Parse("invalid.token")

		require.Error(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("refresh_token_rejected", func(t *testing.T) {
		manager, mockStore, user := setupTestManager(t)
		ctx := context.Background()

		mockStore.On("RevokeAllUserTokens", ctx, user.ID).
			Return(nil).Once()
		mockStore.On("InsertRefreshToken", ctx, mock.AnythingOfType("*models.RefreshToken")).
			Return(nil).Once()

		tokens, err := manager.CreateNewPair(ctx, user)
		require.NoError(t, err)

		_, err = manager.Parse(tokens.Refresh)

		require.Error(t, err)
		mockStore.AssertExpectations(t)
	})
}
