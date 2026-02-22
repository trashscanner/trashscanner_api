package store

import (
	"context"
	"time"

	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) GetAdminUsers(ctx context.Context, limit, offset int32) ([]models.User, error) {
	rows, err := s.q.GetAdminUsers(ctx, db.GetAdminUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, errlocal.NewErrInternal("failed to get admin users list", err.Error(), nil)
	}

	users := make([]models.User, 0, len(rows))
	for _, row := range rows {
		user := models.User{
			ID:        row.ID,
			Login:     row.Login,
			Name:      row.Name,
			Role:      models.Role(row.Role),
			Avatar:    row.Avatar,
			Deleted:   row.Deleted,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}

		if row.LastLoginAt != nil {
			if t, ok := row.LastLoginAt.(time.Time); ok {
				user.LastLoginAt = &t
			}
		}

		if row.Status != nil || row.Rating != nil || row.FilesScanned != nil || row.TotalWeight != nil || row.LastScannedAt.Valid {
			user.Stat = &models.Stat{}
			if row.Status != nil {
				user.Stat.Status = models.UserStatus(*row.Status)
			}
			if row.Rating != nil {
				user.Stat.Rating = int(*row.Rating)
			}
			if row.FilesScanned != nil {
				user.Stat.FilesScanned = int(*row.FilesScanned)
			}
			if row.TotalWeight != nil {
				user.Stat.TotalWeight = *row.TotalWeight
			}
			if row.LastScannedAt.Valid {
				user.Stat.LastScannedAt = row.LastScannedAt.Time
			}
		}

		users = append(users, user)
	}

	return users, nil
}

func (s *pgStore) CountUsers(ctx context.Context) (int64, error) {
	count, err := s.q.CountUsers(ctx)
	if err != nil {
		return 0, errlocal.NewErrInternal("failed to count users", err.Error(), nil)
	}

	return count, nil
}
