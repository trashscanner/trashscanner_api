package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) InsertLoginHistory(
	ctx context.Context,
	loginHistory *models.LoginHistory,
) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	id, err := s.q.CreateLoginHistory(ctx, db.CreateLoginHistoryParams{
		UserID:        loginHistory.UserID,
		Success:       loginHistory.Success,
		FailureReason: loginHistory.FailureReason,
		IpAddress:     loginHistory.IpAddress,
		UserAgent:     loginHistory.UserAgent,
		Location:      loginHistory.Location,
	})
	if err != nil {
		return errlocal.NewErrInternal("failed to create login history", err.Error(),
			map[string]any{"user_id": loginHistory.UserID})
	}
	loginHistory.ID = id

	return nil
}

func (s *pgStore) GetLoginHistory(ctx context.Context, userID uuid.UUID) ([]models.LoginHistory, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	dbHistory := []db.LoginHistory{}
	offset := int32(0)
	for {
		loginHistories, err := s.q.GetLoginHistoryByUser(ctx, db.GetLoginHistoryByUserParams{
			UserID: userID,
			Limit:  defaultQueryLimit,
			Offset: offset,
		})
		if err != nil {
			return nil, errlocal.NewErrInternal("failed to get login history", err.Error(),
				map[string]any{"user_id": userID, "offset": offset})
		}
		dbHistory = append(dbHistory, loginHistories...)

		if len(loginHistories) < defaultQueryLimit || len(loginHistories) == 0 {
			break
		}
		offset += defaultQueryLimit
	}

	res := make([]models.LoginHistory, len(dbHistory))
	for i := range dbHistory {
		res[i] = models.LoginHistory(dbHistory[i])
	}

	return res, nil
}
