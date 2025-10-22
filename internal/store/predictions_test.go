package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func TestCreatePredict(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()
	scanURL := "http://example.com/scan"

	predID := uuid.New()
	mockQ.EXPECT().CreateNewPrediction(mock.Anything, db.CreateNewPredictionParams{
		UserID:    userID,
		TrashScan: scanURL,
		Status:    models.PredictionProcessingStatus.String(),
	}).Return(predID, nil).Once()

	res, err := store.StartPrediction(ctx, userID, scanURL)
	assert.NoError(t, err)
	assert.Equal(t, &predID, res)
}

func TestCompletePrediction(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	result := models.Plastic

	mockQ.EXPECT().CompletePrediction(mock.Anything, db.CompletePredictionParams{
		ID:     predictionID,
		Result: result.StringPtr(),
		Status: models.PredictionCompletedStatus.String(),
	}).Return(nil).Once()

	err := store.CompletePrediction(ctx, predictionID, result)
	assert.NoError(t, err)
}

func TestGetPrediction(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	userID := uuid.New()
	scanURL := "http://example.com/scan.jpg"
	result := "plastic"

	dbPred := db.Prediction{
		ID:        predictionID,
		UserID:    userID,
		TrashScan: scanURL,
		Result:    &result,
		Status:    models.PredictionCompletedStatus.String(),
	}

	mockQ.EXPECT().GetPrediction(mock.Anything, predictionID).Return(dbPred, nil).Once()

	pred, err := store.GetPrediction(ctx, predictionID)
	assert.NoError(t, err)
	assert.Equal(t, predictionID, pred.ID)
	assert.Equal(t, userID, pred.User_id)
	assert.Equal(t, scanURL, pred.Trash_scan)
	assert.Equal(t, models.Plastic, pred.Result)
}

func TestGetPredictionsByUserID(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()
	predictionID := uuid.New()
	scanURL := "http://example.com/scan.jpg"
	result := "plastic"

	dbPred := db.Prediction{
		ID:        predictionID,
		UserID:    userID,
		TrashScan: scanURL,
		Result:    &result,
		Status:    models.PredictionCompletedStatus.String(),
	}

	dbPreds := []db.Prediction{dbPred}

	mockQ.EXPECT().GetPredictionsByUserID(mock.Anything, db.GetPredictionsByUserIDParams{
		UserID: userID,
		Offset: 0,
		Limit:  10,
	}).Return(dbPreds, nil).Once()

	preds, err := store.GetPredictionsByUserID(ctx, userID, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, preds, 1)
	assert.Equal(t, predictionID, preds[0].ID)
	assert.Equal(t, userID, preds[0].User_id)
	assert.Equal(t, scanURL, preds[0].Trash_scan)
	assert.Equal(t, models.Plastic, preds[0].Result)
}

func TestStartPrediction_DuplicateError(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()
	scanURL := "http://example.com/scan"

	mockQ.EXPECT().CreateNewPrediction(mock.Anything, db.CreateNewPredictionParams{
		UserID:    userID,
		TrashScan: scanURL,
		Status:    models.PredictionProcessingStatus.String(),
	}).Return(uuid.Nil, errors.New("pq: duplicate key value violates unique constraint \"predictions_user_id_trash_scan_key\" (SQLSTATE 23505)")).Once()

	res, err := store.StartPrediction(ctx, userID, scanURL)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "prediction for this scan already exists")
}

func TestStartPrediction_DatabaseError(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()
	scanURL := "http://example.com/scan"

	mockQ.EXPECT().CreateNewPrediction(mock.Anything, db.CreateNewPredictionParams{
		UserID:    userID,
		TrashScan: scanURL,
		Status:    models.PredictionProcessingStatus.String(),
	}).Return(uuid.Nil, errors.New("connection refused")).Once()

	res, err := store.StartPrediction(ctx, userID, scanURL)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "database error")
}

func TestCompletePrediction_WithError(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	resultErr := errors.New("prediction failed")

	mockQ.EXPECT().CompletePrediction(mock.Anything, db.CompletePredictionParams{
		ID:     predictionID,
		Error:  stringPtr("prediction failed"),
		Status: models.PredictionFailedStatus.String(),
	}).Return(nil).Once()

	err := store.CompletePrediction(ctx, predictionID, resultErr)
	assert.NoError(t, err)
}

func TestCompletePrediction_InvalidResultType(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	invalidResult := 123

	err := store.CompletePrediction(ctx, predictionID, invalidResult)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid prediction result type")
}

func TestCompletePrediction_DatabaseError(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	result := models.Plastic

	mockQ.EXPECT().CompletePrediction(mock.Anything, db.CompletePredictionParams{
		ID:     predictionID,
		Result: result.StringPtr(),
		Status: models.PredictionCompletedStatus.String(),
	}).Return(errors.New("connection refused")).Once()

	err := store.CompletePrediction(ctx, predictionID, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestGetPrediction_NotFound(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()

	mockQ.EXPECT().GetPrediction(mock.Anything, predictionID).Return(db.Prediction{}, sql.ErrNoRows).Once()

	pred, err := store.GetPrediction(ctx, predictionID)
	assert.Error(t, err)
	assert.Nil(t, pred)
	assert.Contains(t, err.Error(), "prediction not found")
}

func TestGetPredictionsByUserID_EmptyList(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()

	mockQ.EXPECT().GetPredictionsByUserID(mock.Anything, db.GetPredictionsByUserIDParams{
		UserID: userID,
		Offset: 0,
		Limit:  10,
	}).Return([]db.Prediction{}, nil).Once()

	preds, err := store.GetPredictionsByUserID(ctx, userID, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, preds, 0)
}

func TestGetPredictionsByUserID_DatabaseError(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()

	mockQ.EXPECT().GetPredictionsByUserID(mock.Anything, db.GetPredictionsByUserIDParams{
		UserID: userID,
		Offset: 0,
		Limit:  10,
	}).Return([]db.Prediction(nil), errors.New("connection refused")).Once()

	preds, err := store.GetPredictionsByUserID(ctx, userID, 0, 10)
	assert.Error(t, err)
	assert.Nil(t, preds)
	assert.Contains(t, err.Error(), "database error")
}

func stringPtr(s string) *string {
	return &s
}
