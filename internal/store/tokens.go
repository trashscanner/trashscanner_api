package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) InsertRefreshToken(
	ctx context.Context,
	refreshToken *models.RefreshToken,
) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	id, err := s.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    refreshToken.UserID,
		TokenHash: refreshToken.TokenHash,
		ExpiresAt: refreshToken.ExpiresAt,
	})
	if err != nil {
		return errlocal.NewErrInternal("failed to create refresh token", err.Error(),
			map[string]any{"user_id": refreshToken.UserID})
	}
	refreshToken.ID = id

	return nil
}

func (s *pgStore) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	rt, err := s.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errlocal.NewErrNotFound("refresh token not found", "no token with given hash",
				map[string]any{"token_hash": tokenHash})
		}
		return nil, errlocal.NewErrInternal("failed to get refresh token", err.Error(),
			map[string]any{"token_hash": tokenHash})
	}
	model := models.RefreshToken(rt)

	return &model, nil
}

func (s *pgStore) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	err := s.q.RevokeRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errlocal.NewErrNotFound("refresh token not found", "no token to revoke",
				map[string]any{"token_hash": tokenHash})
		}
		return errlocal.NewErrInternal("failed to revoke refresh token", err.Error(),
			map[string]any{"token_hash": tokenHash})
	}
	return nil
}

func (s *pgStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	err := s.q.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		return errlocal.NewErrInternal("failed to revoke all user tokens", err.Error(),
			map[string]any{"user_id": userID})
	}
	return nil
}
