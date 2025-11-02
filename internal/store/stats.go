package store

import (
	"context"
	"encoding/json"

	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) UpdateStats(ctx context.Context, stat *models.Stat) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	rawTrashByTypes, err := json.Marshal(stat.TrashByTypes)
	if err != nil {
		return errlocal.NewErrInternal("failed to marshal user stats", err.Error(),
			map[string]any{"user_id": stat.ID.String()})
	}

	err = s.q.UpdateStats(ctx, db.UpdateStatsParams{
		ID:           stat.ID,
		Status:       string(stat.Status),
		Rating:       int32(stat.Rating),
		FilesScanned: int32(stat.FilesScanned),
		TotalWeight:  float64(stat.TotalWeight),
		TrashByTypes: rawTrashByTypes,
	})
	if err != nil {
		return errlocal.NewErrInternal("failed to update user stats", err.Error(),
			map[string]any{"user_id": stat.ID.String()})
	}
	return nil
}
