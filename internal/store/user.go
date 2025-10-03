package store

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) CreateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	_, gErr := s.q.GetUserByLogin(ctx, user.Login)
	if gErr == nil {
		return fmt.Errorf("user with login %s already exists", user.Login)
	}
	if !errors.Is(gErr, pgx.ErrNoRows) {
		return fmt.Errorf("database error: %w", gErr)
	}

	tx, err := s.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, mErr := s.WithTx(tx).CreateUser(ctx, db.CreateUserParams{
		Login:          user.Login,
		HashedPassword: user.HashedPassword,
	})
	if mErr != nil {
		return mErr
	}
	user.ID = id

	if _, err := s.WithTx(tx).CreateStats(ctx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *pgStore) GetUser(ctx context.Context, id uuid.UUID, withStats bool) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	dbUser, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var user models.User
	user.Model(dbUser)

	if withStats {
		stats, err := s.q.GetStatsByUserID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get user stats: %w", err)
		}
		user.WithStat(stats)
	}

	return &user, nil
}

func (s *pgStore) UpdateUserPass(ctx context.Context, id uuid.UUID, newHashedPass string) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	return s.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:             id,
		HashedPassword: newHashedPass,
	})
}

func (s *pgStore) UpdateAvatar(ctx context.Context, id uuid.UUID, avatarURL string) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	if _, err := url.Parse(avatarURL); err != nil {
		return fmt.Errorf("invalid avatar URL: %w", err)
	}

	return s.q.UpdateUserAvatar(ctx, db.UpdateUserAvatarParams{
		ID:     id,
		Avatar: &avatarURL,
	})
}

func (s *pgStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	return s.q.DeleteUser(ctx, id)
}
