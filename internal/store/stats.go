package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) UpdateStats(ctx context.Context, userID uuid.UUID, stat models.Stat) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	return s.q.UpdateStats(ctx, db.UpdateStatsParams{
		UserID:       userID,
		Status:       string(stat.Status),
		Rating:       int32(stat.Rating),
		FilesScanned: int32(stat.FilesScanned),
		TotalWeight:  float64(stat.TotalWeight),
	})
}
