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

func (s *pgStore) CreateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	_, gErr := s.q.GetUserByLogin(ctx, user.Login)
	if gErr == nil {
		return errlocal.NewErrConflict("user with this login already exists", "login already taken",
			map[string]any{"login": user.Login})
	}
	if !errors.Is(gErr, pgx.ErrNoRows) {
		return errlocal.NewErrInternal("failed to check existing user", gErr.Error(),
			map[string]any{"login": user.Login})
	}

	id, cErr := s.q.CreateUser(ctx, db.CreateUserParams{
		Login:          user.Login,
		HashedPassword: user.HashedPassword,
	})
	if cErr != nil {
		return errlocal.NewErrInternal("failed to create user", cErr.Error(),
			map[string]any{"login": user.Login})
	}
	user.ID = id

	return nil
}

func (s *pgStore) GetUser(ctx context.Context, id uuid.UUID, withStats bool) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	dbUser, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errlocal.NewErrNotFound("user not found", "no user with given ID",
				map[string]any{"user_id": id})
		}
		return nil, errlocal.NewErrInternal("failed to get user", err.Error(),
			map[string]any{"user_id": id})
	}

	var user models.User
	user.Model(dbUser)

	if withStats {
		stats, err := s.q.GetStatsByUserID(ctx, id)
		if err != nil {
			return nil, errlocal.NewErrInternal("failed to get user stats", err.Error(),
				map[string]any{"user_id": id})
		}
		user.WithStat(stats)
	}

	return &user, nil
}

func (s *pgStore) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	dbUser, err := s.q.GetUserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errlocal.NewErrNotFound("user not found", "no user with given login",
				map[string]any{"login": login})
		}
		return nil, errlocal.NewErrInternal("failed to get user by login", err.Error(),
			map[string]any{"login": login})
	}

	var user models.User
	user.Model(dbUser)

	return &user, nil
}

func (s *pgStore) UpdateUser(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	if err := s.q.UpdateUser(ctx, db.UpdateUserParams{
		ID:   user.ID,
		Name: user.Name,
	}); err != nil {
		return errlocal.NewErrInternal("failed to update user", err.Error(),
			map[string]any{"user_id": user.ID})
	}

	return nil
}

func (s *pgStore) UpdateUserPass(ctx context.Context, id uuid.UUID, newHashedPass string) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	if err := s.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:             id,
		HashedPassword: newHashedPass,
	}); err != nil {
		return errlocal.NewErrInternal("failed to update user password", err.Error(),
			map[string]any{"user_id": id})
	}

	return nil
}

func (s *pgStore) UpdateAvatar(ctx context.Context, user *models.User) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	if err := s.q.UpdateUserAvatar(ctx, db.UpdateUserAvatarParams{
		ID:     user.ID,
		Avatar: user.Avatar,
	}); err != nil {
		return errlocal.NewErrInternal("failed to update user avatar", err.Error(),
			map[string]any{"user_id": user.ID.String()})
	}

	return nil
}

func (s *pgStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	if err := s.q.DeleteUser(ctx, id); err != nil {
		return errlocal.NewErrInternal("failed to delete user", err.Error(),
			map[string]any{"user_id": id})
	}

	return nil
}
