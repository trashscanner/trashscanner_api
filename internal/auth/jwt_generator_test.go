package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestNewJWTGenerator(t *testing.T) {
	t.Run("success", func(t *testing.T) {
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
		assert.NotNil(t, generator)
		assert.NotNil(t, generator.privateKey)
		assert.NotNil(t, generator.publicKey)
		assert.Equal(t, 15*time.Minute, generator.ttlAccess)
		assert.Equal(t, 7*24*time.Hour, generator.ttlRefresh)
	})

	t.Run("missing_keys", func(t *testing.T) {
		t.Setenv("AUTH_MANAGER_SECRET_PRIVATE_KEY", "")
		t.Setenv("AUTH_MANAGER_PUBLIC_KEY", "")

		cfg := config.Config{
			Auth: config.AuthManagerConfig{
				Algorithm: "EdDSA",
			},
		}

		_, err := newJWTGenerator(cfg)

		require.Error(t, err)
	})
}

func TestJWTGenerator_NewPair(t *testing.T) {
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

	user := models.User{
		ID:    uuid.New(),
		Login: "testuser",
	}

	t.Run("success", func(t *testing.T) {
		tokens, err := generator.newPair(user)

		require.NoError(t, err)
		assert.NotEmpty(t, tokens.Access)
		assert.NotEmpty(t, tokens.Refresh)
	})

	t.Run("different_tokens", func(t *testing.T) {
		tokens, err := generator.newPair(user)

		require.NoError(t, err)
		assert.NotEqual(t, tokens.Access, tokens.Refresh)
	})

	t.Run("unique_tokens_each_call", func(t *testing.T) {
		tokens1, err := generator.newPair(user)
		require.NoError(t, err)

		tokens2, err := generator.newPair(user)
		require.NoError(t, err)

		assert.NotEqual(t, tokens1.Access, tokens2.Access)
		assert.NotEqual(t, tokens1.Refresh, tokens2.Refresh)
	})
}

func TestJWTGenerator_ParseAccess(t *testing.T) {
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

	user := models.User{
		ID:    uuid.New(),
		Login: "testuser",
	}

	t.Run("valid_token", func(t *testing.T) {
		tokens, err := generator.newPair(user)
		require.NoError(t, err)

		claims, err := generator.parseAccess(tokens.Access)

		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), claims.UserID)
		assert.Equal(t, user.Login, claims.Login)
		assert.Equal(t, "access", claims.TokenType)
	})

	t.Run("reject_refresh_token", func(t *testing.T) {
		tokens, err := generator.newPair(user)
		require.NoError(t, err)

		_, err = generator.parseAccess(tokens.Refresh)

		require.Error(t, err)
	})

	t.Run("invalid_token", func(t *testing.T) {
		_, err := generator.parseAccess("invalid.token.here")

		require.Error(t, err)
	})

	t.Run("empty_token", func(t *testing.T) {
		_, err := generator.parseAccess("")

		require.Error(t, err)
	})

	t.Run("wrong_signature", func(t *testing.T) {
		utils.GenerateAndSetKeys()
		wrongGenerator, err := newJWTGenerator(cfg)
		require.NoError(t, err)

		tokens, err := wrongGenerator.newPair(user)
		require.NoError(t, err)

		_, err = generator.parseAccess(tokens.Access)

		require.Error(t, err)
	})

	t.Run("expired_token", func(t *testing.T) {
		utils.GenerateAndSetKeys()
		shortCfg := config.Config{
			Auth: config.AuthManagerConfig{
				Algorithm:       "EdDSA",
				AccessTokenTTL:  1 * time.Millisecond,
				RefreshTokenTTL: 1 * time.Second,
			},
		}
		shortGenerator, err := newJWTGenerator(shortCfg)
		require.NoError(t, err)

		tokens, err := shortGenerator.newPair(user)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		_, err = shortGenerator.parseAccess(tokens.Access)

		require.Error(t, err)
	})
}

func TestJWTGenerator_ParseRefresh(t *testing.T) {
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

	user := models.User{
		ID:    uuid.New(),
		Login: "testuser",
	}

	t.Run("valid_token", func(t *testing.T) {
		tokens, err := generator.newPair(user)
		require.NoError(t, err)

		token, err := generator.parseRefresh(tokens.Refresh)

		require.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, user.ID, token.UserID)
		assert.NotEmpty(t, token.TokenHash)
	})

	t.Run("reject_access_token", func(t *testing.T) {
		tokens, err := generator.newPair(user)
		require.NoError(t, err)

		_, err = generator.parseRefresh(tokens.Access)

		require.Error(t, err)
	})

	t.Run("invalid_token", func(t *testing.T) {
		_, err := generator.parseRefresh("invalid.token.here")

		require.Error(t, err)
	})

	t.Run("empty_token", func(t *testing.T) {
		_, err := generator.parseRefresh("")

		require.Error(t, err)
	})

	t.Run("wrong_signature", func(t *testing.T) {
		utils.GenerateAndSetKeys()
		wrongGenerator, err := newJWTGenerator(cfg)
		require.NoError(t, err)

		tokens, err := wrongGenerator.newPair(user)
		require.NoError(t, err)

		_, err = generator.parseRefresh(tokens.Refresh)

		require.Error(t, err)
	})

	t.Run("expired_token", func(t *testing.T) {
		utils.GenerateAndSetKeys()
		shortCfg := config.Config{
			Auth: config.AuthManagerConfig{
				Algorithm:       "EdDSA",
				AccessTokenTTL:  1 * time.Second,
				RefreshTokenTTL: 1 * time.Millisecond,
			},
		}
		shortGenerator, err := newJWTGenerator(shortCfg)
		require.NoError(t, err)

		tokens, err := shortGenerator.newPair(user)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		_, err = shortGenerator.parseRefresh(tokens.Refresh)

		require.Error(t, err)
	})

	t.Run("token_hash_consistency", func(t *testing.T) {
		tokens, err := generator.newPair(user)
		require.NoError(t, err)

		token1, err := generator.parseRefresh(tokens.Refresh)
		require.NoError(t, err)

		token2, err := generator.parseRefresh(tokens.Refresh)
		require.NoError(t, err)

		assert.Equal(t, token1.TokenHash, token2.TokenHash)
	})
}
