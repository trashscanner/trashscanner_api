package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func (s *pgStore) StartPrediction(ctx context.Context, userID uuid.UUID, scanURL string) (*models.Prediction, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	prediction, err := s.q.CreateNewPrediction(ctx, db.CreateNewPredictionParams{
		UserID:    userID,
		TrashScan: scanURL,
		Status:    models.PredictionProcessingStatus.String(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "SQLSTATE 23505") {
			return nil, errlocal.NewErrConflict(
				"prediction for this scan already exists",
				err.Error(),
				map[string]any{"user_id": userID.String(), "scan": scanURL},
			)
		}

		return nil, errlocal.NewErrInternal(
			"database error",
			err.Error(),
			map[string]any{"user_id": userID.String(), "scan": scanURL},
		)
	}

	model := new(models.Prediction)
	model.Model(prediction)

	return model, nil
}

func (s *pgStore) CompletePrediction(
	ctx context.Context,
	id uuid.UUID,
	result models.PredictionResult,
	err error,
) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	params := db.CompletePredictionParams{ID: id}
	if err != nil {
		params.Error = utils.Ptr(err.Error())
		params.Status = models.PredictionFailedStatus.String()
	} else {
		raw, err := json.Marshal(result)
		if err != nil {
			return errlocal.NewErrInternal("failed to marshal prediction result", err.Error(), nil)
		}
		params.Result = raw
		params.Status = models.PredictionCompletedStatus.String()
	}

	if dbErr := s.q.CompletePrediction(ctx, params); dbErr != nil {
		return errlocal.NewErrInternal("database error", dbErr.Error(), nil)
	}

	return nil
}

func (s *pgStore) GetPrediction(ctx context.Context, id uuid.UUID) (*models.Prediction, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	dbPrediction, err := s.q.GetPrediction(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errlocal.NewErrNotFound(
				"prediction not found",
				err.Error(),
				map[string]any{"prediction_id": id.String()},
			)
		}

		return nil, errlocal.NewErrInternal("database error", err.Error(), nil)
	}

	model := &models.Prediction{}
	model.Model(dbPrediction)

	return model, nil
}

func (s *pgStore) GetPredictionsByUserID(
	ctx context.Context,
	userID uuid.UUID,
	offset, limit int,
) ([]*models.Prediction, error) {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	predictions, err := s.q.GetPredictionsByUserID(ctx, db.GetPredictionsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, errlocal.NewErrInternal("database error", err.Error(), nil)
	}

	return models.NewPredictionsList(predictions), nil
}
