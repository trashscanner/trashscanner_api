package store

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func TestPgStore_GetAdminUsers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		now := time.Now()
		userID := uuid.New()

		statusStr := string(models.UserStatusNewbie)
		rating := int32(100)
		filesScanned := int32(50)
		totalWeight := float64(10.5)

		rows := []db.GetAdminUsersRow{
			{
				ID:           userID,
				Login:        "admin",
				Name:         "Admin User",
				Role:         "admin",
				Avatar:       nil,
				Deleted:      false,
				CreatedAt:    now,
				UpdatedAt:    now,
				LastLoginAt:  now,
				Status:       &statusStr,
				Rating:       &rating,
				FilesScanned: &filesScanned,
				TotalWeight:  &totalWeight,
				LastScannedAt: pgtype.Timestamptz{
					Time:  now,
					Valid: true,
				},
			},
			{
				ID:        uuid.New(),
				Login:     "user_no_stats",
				Name:      "User No Stats",
				Role:      "user",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		q.EXPECT().GetAdminUsers(mock.Anything, db.GetAdminUsersParams{
			Limit:  10,
			Offset: 0,
		}).Return(rows, nil)

		users, err := store.GetAdminUsers(context.Background(), 10, 0)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		// Check first user with stats
		assert.Equal(t, userID, users[0].ID)
		assert.Equal(t, "admin", users[0].Login)
		assert.NotNil(t, users[0].LastLoginAt)
		assert.NotNil(t, users[0].Stat)
		assert.Equal(t, models.UserStatusNewbie, users[0].Stat.Status)
		assert.Equal(t, 100, users[0].Stat.Rating)
		assert.Equal(t, 50, users[0].Stat.FilesScanned)
		assert.Equal(t, 10.5, users[0].Stat.TotalWeight)
		assert.Equal(t, now, users[0].Stat.LastScannedAt)

		// Check second user without stats
		assert.Equal(t, "user_no_stats", users[1].Login)
		assert.Nil(t, users[1].LastLoginAt)
		assert.Nil(t, users[1].Stat)
	})

	t.Run("error", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		q.EXPECT().GetAdminUsers(mock.Anything, mock.AnythingOfType("db.GetAdminUsersParams")).
			Return(nil, sql.ErrConnDone)

		users, err := store.GetAdminUsers(context.Background(), 10, 0)
		assert.Error(t, err)
		assert.Nil(t, users)
	})
}

func TestPgStore_GetAdminUserByID(t *testing.T) {
	t.Run("success_with_stats", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		now := time.Now()
		userID := uuid.New()

		statusStr := string(models.UserStatusNewbie)
		rating := int32(100)
		filesScanned := int32(50)
		totalWeight := float64(10.5)

		row := db.GetAdminUserByIDRow{
			ID:           userID,
			Login:        "testuser",
			Name:         "Test User",
			Role:         "user",
			Avatar:       nil,
			Deleted:      false,
			CreatedAt:    now,
			UpdatedAt:    now,
			LastLoginAt:  now,
			Status:       &statusStr,
			Rating:       &rating,
			FilesScanned: &filesScanned,
			TotalWeight:  &totalWeight,
			LastScannedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
		}

		q.EXPECT().GetAdminUserByID(mock.Anything, userID).Return(row, nil)

		user, err := store.GetAdminUserByID(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, userID, user.ID)
		assert.Equal(t, "testuser", user.Login)
		assert.Equal(t, "Test User", user.Name)
		assert.Equal(t, models.Role("user"), user.Role)
		assert.NotNil(t, user.LastLoginAt)
		assert.NotNil(t, user.Stat)
		assert.Equal(t, models.UserStatusNewbie, user.Stat.Status)
		assert.Equal(t, 100, user.Stat.Rating)
		assert.Equal(t, 50, user.Stat.FilesScanned)
		assert.Equal(t, 10.5, user.Stat.TotalWeight)
		assert.Equal(t, now, user.Stat.LastScannedAt)
	})

	t.Run("success_without_stats", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		now := time.Now()
		userID := uuid.New()

		row := db.GetAdminUserByIDRow{
			ID:        userID,
			Login:     "nostatuser",
			Name:      "No Stat User",
			Role:      "user",
			CreatedAt: now,
			UpdatedAt: now,
		}

		q.EXPECT().GetAdminUserByID(mock.Anything, userID).Return(row, nil)

		user, err := store.GetAdminUserByID(context.Background(), userID)
		require.NoError(t, err)
		require.NotNil(t, user)

		assert.Equal(t, "nostatuser", user.Login)
		assert.Nil(t, user.LastLoginAt)
		assert.Nil(t, user.Stat)
	})

	t.Run("not_found", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		userID := uuid.New()

		q.EXPECT().GetAdminUserByID(mock.Anything, userID).
			Return(db.GetAdminUserByIDRow{}, fmt.Errorf("no rows in result set"))

		user, err := store.GetAdminUserByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("db_error", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		userID := uuid.New()

		q.EXPECT().GetAdminUserByID(mock.Anything, userID).
			Return(db.GetAdminUserByIDRow{}, sql.ErrConnDone)

		user, err := store.GetAdminUserByID(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestPgStore_CountUsers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		q.EXPECT().CountUsers(mock.Anything).Return(int64(42), nil)

		count, err := store.CountUsers(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(42), count)
	})

	t.Run("error", func(t *testing.T) {
		q := mocks.NewQuerier(t)
		store := &pgStore{q: q}

		q.EXPECT().CountUsers(mock.Anything).Return(int64(0), sql.ErrConnDone)

		count, err := store.CountUsers(context.Background())
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
	})
}
