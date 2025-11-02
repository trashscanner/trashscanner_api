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
	}).Return(db.Prediction{ID: predID}, nil).Once()

	res, err := store.StartPrediction(ctx, userID, scanURL)
	assert.NoError(t, err)
	assert.Equal(t, predID, res.ID)
}

func TestCompletePrediction(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	result := models.PredictionResult{"plastic": 0.95, "metal": 0.05}

	expectedJSON := []byte(`{"metal":0.05,"plastic":0.95}`)

	mockQ.EXPECT().CompletePrediction(mock.Anything, mock.MatchedBy(func(params db.CompletePredictionParams) bool {
		return params.ID == predictionID &&
			params.Status == models.PredictionCompletedStatus.String() &&
			string(params.Result) == string(expectedJSON)
	})).Return(nil).Once()

	err := store.CompletePrediction(ctx, predictionID, result, nil)
	assert.NoError(t, err)
}

func TestGetPrediction(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	userID := uuid.New()
	scanURL := "http://example.com/scan.jpg"
	result := []byte(`{"plastic":0.95,"metal":0.05}`)

	dbPred := db.Prediction{
		ID:        predictionID,
		UserID:    userID,
		TrashScan: scanURL,
		Result:    result,
		Status:    models.PredictionCompletedStatus.String(),
	}

	mockQ.EXPECT().GetPrediction(mock.Anything, predictionID).Return(dbPred, nil).Once()

	pred, err := store.GetPrediction(ctx, predictionID)
	assert.NoError(t, err)
	assert.Equal(t, predictionID, pred.ID)
	assert.Equal(t, userID, pred.UserID)
	assert.Equal(t, scanURL, pred.TrashScan)
	assert.NotNil(t, pred.Result)
	assert.Equal(t, 0.95, pred.Result["plastic"])
	assert.Equal(t, 0.05, pred.Result["metal"])
}

func TestGetPredictionsByUserID(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	userID := uuid.New()
	predictionID := uuid.New()
	scanURL := "http://example.com/scan.jpg"
	result := []byte(`{"plastic":0.95,"metal":0.05}`)

	dbPred := db.Prediction{
		ID:        predictionID,
		UserID:    userID,
		TrashScan: scanURL,
		Result:    result,
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
	assert.Equal(t, userID, preds[0].UserID)
	assert.Equal(t, scanURL, preds[0].TrashScan)
	assert.NotNil(t, preds[0].Result)
	assert.Equal(t, 0.95, preds[0].Result["plastic"])
	assert.Equal(t, 0.05, preds[0].Result["metal"])
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
	}).Return(db.Prediction{}, errors.New("pq: duplicate key value violates unique constraint \"predictions_user_id_trash_scan_key\" (SQLSTATE 23505)")).Once()

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
	}).Return(db.Prediction{}, errors.New("connection refused")).Once()

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

	err := store.CompletePrediction(ctx, predictionID, nil, resultErr)
	assert.NoError(t, err)
}

func TestCompletePrediction_DatabaseError(t *testing.T) {
	ctx := context.Background()

	mockQ := dbMock.NewQuerier(t)
	store := &pgStore{q: mockQ}
	predictionID := uuid.New()
	result := models.PredictionResult{"plastic": 0.95}

	mockQ.EXPECT().CompletePrediction(mock.Anything, mock.MatchedBy(func(params db.CompletePredictionParams) bool {
		return params.ID == predictionID &&
			params.Status == models.PredictionCompletedStatus.String()
	})).Return(errors.New("connection refused")).Once()

	err := store.CompletePrediction(ctx, predictionID, result, nil)
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
