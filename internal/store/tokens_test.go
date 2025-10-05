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
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
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
			UserID:    testdata.User1ID,
			TokenHash: "new_token_hash",
			ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}

		expectedID := uuid.New()

		mockQ.EXPECT().CreateRefreshToken(mock.Anything, db.CreateRefreshTokenParams{
			UserID:    testdata.User1ID,
			TokenHash: "new_token_hash",
			ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}).Return(expectedID, nil).Once()

		err := store.InsertRefreshToken(ctx, &newToken)

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
			UserID:    testdata.User1ID,
			TokenHash: "new_token_hash",
			ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}

		dbErr := assert.AnError

		mockQ.EXPECT().CreateRefreshToken(mock.Anything, mock.Anything).
			Return(uuid.Nil, dbErr).Once()

		err := store.InsertRefreshToken(ctx, &newToken)

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), dbErr.Error())
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

		assert.Error(t, err)
		assert.Nil(t, token)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), dbErr.Error())
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

		assert.Error(t, err)
		assert.Nil(t, token)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), dbErr.Error())
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

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), dbErr.Error())
	})

	t.Run("Revoke all user tokens when user has no tokens", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		mockQ.EXPECT().RevokeAllUserTokens(mock.Anything, testdata.User2ID).
			Return(nil).Once()

		err := store.RevokeAllUserTokens(ctx, testdata.User2ID)

		assert.NoError(t, err)
	})
}
