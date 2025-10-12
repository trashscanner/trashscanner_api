package store

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	testdata "github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestInsertRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)

		s := &pgStore{q: mockQ}

		newToken := models.RefreshToken{
			UserID:    testdata.User1ID,
			TokenHash: "new_token_hash",
			ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}

		expectedID := uuid.New()

		mockQ.EXPECT().
			CreateRefreshToken(mock.Anything, db.CreateRefreshTokenParams{
				UserID:    newToken.UserID,
				TokenHash: newToken.TokenHash,
				ExpiresAt: newToken.ExpiresAt,
			}).
			Return(expectedID, nil).
			Once()

		err := s.InsertRefreshToken(ctx, &newToken)

		assert.NoError(t, err)
		assert.Equal(t, expectedID, newToken.ID)
	})

	t.Run("database error", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)

		s := &pgStore{q: mockQ}

		newToken := models.RefreshToken{
			UserID:    testdata.User1ID,
			TokenHash: "new_token_hash",
			ExpiresAt: time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC),
		}

		dbErr := errors.New("db failure")

		mockQ.EXPECT().
			CreateRefreshToken(mock.Anything, mock.AnythingOfType("db.CreateRefreshTokenParams")).
			Return(uuid.Nil, dbErr).
			Once()

		err := s.InsertRefreshToken(ctx, &newToken)

		var internalErr *errlocal.ErrInternal
		require.ErrorAs(t, err, &internalErr)
		assert.Contains(t, internalErr.Error(), dbErr.Error())
	})
}

func TestGetRefreshTokenByHash(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		expected := db.RefreshToken(testdata.RefreshToken1)

		mockQ.EXPECT().
			GetRefreshTokenByHash(mock.Anything, expected.TokenHash).
			Return(expected, nil).
			Once()

		actual, err := s.GetRefreshTokenByHash(ctx, expected.TokenHash)

		assert.NoError(t, err)
		require.NotNil(t, actual)
		assert.Equal(t, expected.TokenHash, actual.TokenHash)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		hash := "missing-token"

		mockQ.EXPECT().
			GetRefreshTokenByHash(mock.Anything, hash).
			Return(db.RefreshToken{}, pgx.ErrNoRows).
			Once()

		_, err := s.GetRefreshTokenByHash(ctx, hash)

		var notFoundErr *errlocal.ErrNotFound
		require.ErrorAs(t, err, &notFoundErr)
		assert.Equal(t, http.StatusNotFound, notFoundErr.Code())
	})

	t.Run("database error", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		hash := "hash_token_1"
		dbErr := errors.New("unavailable")

		mockQ.EXPECT().
			GetRefreshTokenByHash(mock.Anything, hash).
			Return(db.RefreshToken{}, dbErr).
			Once()

		_, err := s.GetRefreshTokenByHash(ctx, hash)

		var internalErr *errlocal.ErrInternal
		require.ErrorAs(t, err, &internalErr)
		assert.Contains(t, internalErr.Error(), dbErr.Error())
	})
}

func TestRevokeRefreshToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		hash := "hash123"

		mockQ.EXPECT().
			RevokeRefreshToken(mock.Anything, hash).
			Return(nil).
			Once()

		err := s.RevokeRefreshToken(ctx, hash)

		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		hash := "missing"

		mockQ.EXPECT().
			RevokeRefreshToken(mock.Anything, hash).
			Return(pgx.ErrNoRows).
			Once()

		err := s.RevokeRefreshToken(ctx, hash)

		var notFoundErr *errlocal.ErrNotFound
		require.ErrorAs(t, err, &notFoundErr)
		assert.Equal(t, http.StatusNotFound, notFoundErr.Code())
	})

	t.Run("database error", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		hash := "hash123"
		dbErr := errors.New("db down")

		mockQ.EXPECT().
			RevokeRefreshToken(mock.Anything, hash).
			Return(dbErr).
			Once()

		err := s.RevokeRefreshToken(ctx, hash)

		var internalErr *errlocal.ErrInternal
		require.ErrorAs(t, err, &internalErr)
		assert.Contains(t, internalErr.Error(), dbErr.Error())
	})
}

func TestRevokeAllUserTokens(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		mockQ.EXPECT().
			RevokeAllUserTokens(mock.Anything, testdata.User1ID).
			Return(nil).
			Once()

		err := s.RevokeAllUserTokens(ctx, testdata.User1ID)

		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		ctx := context.Background()
		mockQ := dbMock.NewQuerier(t)
		s := &pgStore{q: mockQ}

		dbErr := errors.New("db down")

		mockQ.EXPECT().
			RevokeAllUserTokens(mock.Anything, testdata.User1ID).
			Return(dbErr).
			Once()

		err := s.RevokeAllUserTokens(ctx, testdata.User1ID)

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), dbErr.Error())
	})
}
