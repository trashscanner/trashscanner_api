package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) InsertRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	refreshToken *models.RefreshToken,
) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	id, err := s.q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:      userID,
		TokenFamily: refreshToken.TokenFamily,
		TokenHash:   refreshToken.TokenHash,
		ExpiresAt:   refreshToken.ExpiresAt,
	})
	refreshToken.ID = id

	return err
}

func (s *pgStore) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	rt, err := s.q.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	model := models.RefreshToken(rt)

	return &model, nil
}

func (s *pgStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	return s.q.RevokeAllUserTokens(ctx, userID)
}

func (s *pgStore) RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	return s.q.RevokeTokenFamily(ctx, tokenFamily)
}
