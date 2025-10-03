package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func (s *pgStore) InsertLoginHistory(
	ctx context.Context,
	userID uuid.UUID,
	loginHistory *models.LoginHistory,
) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	id, err := s.q.CreateLoginHistory(ctx, db.CreateLoginHistoryParams{
		UserID:        loginHistory.UserID,
		LoginAttempt:  loginHistory.LoginAttempt,
		Success:       loginHistory.Success,
		FailureReason: loginHistory.FailureReason,
		IpAddress:     loginHistory.IpAddress,
		UserAgent:     loginHistory.UserAgent,
		Location:      loginHistory.Location,
	})
	loginHistory.ID = id

	return err
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
			return nil, err
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
