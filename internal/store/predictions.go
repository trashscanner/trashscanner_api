package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
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

// result should be error or models.PredictionResult
func (s *pgStore) CompletePrediction(ctx context.Context, id uuid.UUID, result any) error {
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()

	params := db.CompletePredictionParams{ID: id}

	switch res := result.(type) {
	case error:
		errString := res.Error()
		params.Error = &errString
		params.Status = models.PredictionFailedStatus.String()
	case models.PredictionResult:
		raw, err := json.Marshal(res)
		if err != nil {
			return errlocal.NewErrInternal("failed to marshal prediction result", err.Error(), nil)
		}
		params.Result = raw
		params.Status = models.PredictionCompletedStatus.String()
	default:
		return errlocal.NewErrInternal("invalid prediction result type", "",
			map[string]any{"type": reflect.TypeOf(result).String()})
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
