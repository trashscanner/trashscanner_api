package store

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestInsertRefreshToken(t *testing.T) {
	t.Run("Insert refresh token successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		newToken := models.RefreshToken{
			UserID:      testdata.User1ID,
			TokenFamily: testdata.TokenFamily1,
			TokenHash:   "new_token_hash",
			ExpiresAt:   time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}

		expectedID := uuid.New()

		mockQ.EXPECT().CreateRefreshToken(mock.Anything, db.CreateRefreshTokenParams{
			UserID:      testdata.User1ID,
			TokenFamily: testdata.TokenFamily1,
			TokenHash:   "new_token_hash",
			ExpiresAt:   time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}).Return(expectedID, nil).Once()

		err := store.InsertRefreshToken(ctx, testdata.User1ID, &newToken)

		assert.NoError(t, err)
		assert.Equal(t, expectedID, newToken.ID)
	})

	t.Run("Insert refresh token fails on database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		newToken := models.RefreshToken{
			UserID:      testdata.User1ID,
			TokenFamily: testdata.TokenFamily1,
			TokenHash:   "new_token_hash",
			ExpiresAt:   time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}

		dbErr := assert.AnError

		mockQ.EXPECT().CreateRefreshToken(mock.Anything, mock.Anything).
			Return(uuid.Nil, dbErr).Once()

		err := store.InsertRefreshToken(ctx, testdata.User1ID, &newToken)

		assert.ErrorIs(t, err, dbErr)
	})
}

func TestGetRefreshTokenByHash(t *testing.T) {
	t.Run("Get refresh token by hash successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		tokenHash := "hash_token_1"
		expectedToken := db.RefreshToken(testdata.RefreshToken1)

		mockQ.EXPECT().GetRefreshTokenByHash(mock.Anything, tokenHash).
			Return(expectedToken, nil).Once()

		token, err := store.GetRefreshTokenByHash(ctx, tokenHash)

		assert.NoError(t, err)
		assert.NotNil(t, token)
		assert.Equal(t, testdata.RefreshToken1ID, token.ID)
		assert.Equal(t, testdata.User1ID, token.UserID)
		assert.Equal(t, testdata.TokenFamily1, token.TokenFamily)
		assert.Equal(t, tokenHash, token.TokenHash)
	})

	t.Run("Get refresh token fails when token not found", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		tokenHash := "non_existent_token"
		dbErr := assert.AnError

		mockQ.EXPECT().GetRefreshTokenByHash(mock.Anything, tokenHash).
			Return(db.RefreshToken{}, dbErr).Once()

		token, err := store.GetRefreshTokenByHash(ctx, tokenHash)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, token)
	})

	t.Run("Get refresh token fails on database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		tokenHash := "hash_token_1"
		dbErr := assert.AnError

		mockQ.EXPECT().GetRefreshTokenByHash(mock.Anything, tokenHash).
			Return(db.RefreshToken{}, dbErr).Once()

		token, err := store.GetRefreshTokenByHash(ctx, tokenHash)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, token)
	})
}

func TestRevokeAllUserTokens(t *testing.T) {
	t.Run("Revoke all user tokens successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		mockQ.EXPECT().RevokeAllUserTokens(mock.Anything, testdata.User1ID).
			Return(nil).Once()

		err := store.RevokeAllUserTokens(ctx, testdata.User1ID)

		assert.NoError(t, err)
	})

	t.Run("Revoke all user tokens fails on database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbErr := assert.AnError

		mockQ.EXPECT().RevokeAllUserTokens(mock.Anything, testdata.User1ID).
			Return(dbErr).Once()

		err := store.RevokeAllUserTokens(ctx, testdata.User1ID)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("Revoke all user tokens when user has no tokens", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		// Даже если у пользователя нет токенов, операция должна успешно завершиться
		mockQ.EXPECT().RevokeAllUserTokens(mock.Anything, testdata.User2ID).
			Return(nil).Once()

		err := store.RevokeAllUserTokens(ctx, testdata.User2ID)

		assert.NoError(t, err)
	})
}

func TestRevokeTokenFamily(t *testing.T) {
	t.Run("Revoke token family successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		mockQ.EXPECT().RevokeTokenFamily(mock.Anything, testdata.TokenFamily1).
			Return(nil).Once()

		err := store.RevokeTokenFamily(ctx, testdata.TokenFamily1)

		assert.NoError(t, err)
	})

	t.Run("Revoke token family fails on database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbErr := assert.AnError

		mockQ.EXPECT().RevokeTokenFamily(mock.Anything, testdata.TokenFamily1).
			Return(dbErr).Once()

		err := store.RevokeTokenFamily(ctx, testdata.TokenFamily1)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("Revoke token family with multiple tokens", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		// Должны быть отозваны все токены в семействе
		mockQ.EXPECT().RevokeTokenFamily(mock.Anything, testdata.TokenFamily1).
			Return(nil).Once()

		err := store.RevokeTokenFamily(ctx, testdata.TokenFamily1)

		assert.NoError(t, err)
	})

	t.Run("Revoke non-existent token family", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		nonExistentFamily := uuid.New()

		// Даже если семейство не существует, операция должна успешно завершиться
		mockQ.EXPECT().RevokeTokenFamily(mock.Anything, nonExistentFamily).
			Return(nil).Once()

		err := store.RevokeTokenFamily(ctx, nonExistentFamily)

		assert.NoError(t, err)
	})
}
