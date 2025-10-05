package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestUpdateStats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		store := &pgStore{q: mockQ}

		user := testdata.User1
		stats := *user.Stat
		stats.Rating = 1500
		stats.Status = "Eco Hero"
		stats.FilesScanned = 42
		stats.TotalWeight = 123.45

		mockQ.EXPECT().UpdateStats(mock.Anything, db.UpdateStatsParams{
			UserID:       user.ID,
			Status:       string(stats.Status),
			Rating:       int32(stats.Rating),
			FilesScanned: int32(stats.FilesScanned),
			TotalWeight:  float64(stats.TotalWeight),
		}).Return(nil).Once()

		assert.NoError(t, store.UpdateStats(ctx, user.ID, stats))
	})

	t.Run("database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		store := &pgStore{q: mockQ}

		stats := testdata.Stats1
		dbErr := assert.AnError

		mockQ.EXPECT().UpdateStats(mock.Anything, mock.Anything).
			Return(dbErr).
			Once()

		err := store.UpdateStats(ctx, testdata.User1ID, stats)

		assert.Error(t, err)
		var internalErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &internalErr)
		assert.Contains(t, internalErr.System(), dbErr.Error())
	})
}
