package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestUpdateStats(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}

	user := testdata.User1

	stats := user.Stat
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

	assert.NoError(t, store.UpdateStats(ctx, user.ID, *stats))
}
