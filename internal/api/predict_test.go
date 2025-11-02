package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestStartPrediction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, _, _, fileStoreMock, predictorMock := newTestServer(t)

		user := testdata.User1
		scanData := []byte("fake scan image content")
		formData := createMultipartFormWithField(t, "scan", "trash.jpg", "image/jpeg", scanData)

		fileURL := "user123/scans/scan-id-123"
		prediction := &models.Prediction{
			ID:     uuid.New(),
			UserID: user.ID,
			Status: "processing",
		}

		fileStoreMock.EXPECT().
			UploadScan(mock.Anything, user.ID.String(), mock.Anything).
			Return(fileURL, nil).
			Once()

		predictorMock.EXPECT().
			Predict(mock.Anything, fileURL).
			Return(prediction, nil).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/predictions", formData.body)
		req.Header.Set("Content-Type", formData.contentType)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.startPrediction(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)

		var response models.Prediction
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, prediction.ID, response.ID)
		assert.Equal(t, prediction.Status, response.Status)
	})

	t.Run("invalid multipart form", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		user := testdata.User1

		req := httptest.NewRequest(http.MethodPost, "/api/v1/predictions", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.startPrediction(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var errResp errlocal.BaseError
		err := json.NewDecoder(rr.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Message(), "bad format of file")
	})

	t.Run("file upload fails", func(t *testing.T) {
		server, _, _, fileStoreMock, _ := newTestServer(t)

		user := testdata.User1
		scanData := []byte("fake scan image content")
		formData := createMultipartFormWithField(t, "scan", "trash.jpg", "image/jpeg", scanData)

		fileStoreMock.EXPECT().
			UploadScan(mock.Anything, user.ID.String(), mock.Anything).
			Return("", assert.AnError).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/predictions", formData.body)
		req.Header.Set("Content-Type", formData.contentType)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.startPrediction(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var errResp errlocal.BaseError
		err := json.NewDecoder(rr.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Message(), "failed to upload scan")
	})

	t.Run("predictor fails", func(t *testing.T) {
		server, _, _, fileStoreMock, predictorMock := newTestServer(t)

		user := testdata.User1
		scanData := []byte("fake scan image content")
		formData := createMultipartFormWithField(t, "scan", "trash.jpg", "image/jpeg", scanData)

		fileURL := "user123/scans/scan-id-123"

		fileStoreMock.EXPECT().
			UploadScan(mock.Anything, user.ID.String(), mock.Anything).
			Return(fileURL, nil).
			Once()

		predictorMock.EXPECT().
			Predict(mock.Anything, fileURL).
			Return(nil, errlocal.NewErrToManyRequests("too many predictions in processing")).
			Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/predictions", formData.body)
		req.Header.Set("Content-Type", formData.contentType)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.startPrediction(rr, req)

		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	})
}

func TestGetPrediction(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		predictionID := uuid.New()
		prediction := &models.Prediction{
			ID:     predictionID,
			UserID: testdata.User1.ID,
			Status: models.PredictionCompletedStatus,
			Result: models.PredictionResult{"plastic": 0.95},
		}

		storeMock.EXPECT().
			GetPrediction(mock.Anything, predictionID).
			Return(prediction, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions/"+predictionID.String(), nil)
		req = mux.SetURLVars(req, map[string]string{predictionIDTag: predictionID.String()})

		rr := httptest.NewRecorder()
		server.getPrediction(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Prediction
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, predictionID, response.ID)
		assert.Equal(t, models.PredictionCompletedStatus, response.Status)
	})

	t.Run("success with failed prediction", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		predictionID := uuid.New()
		predictionError := errlocal.NewErrInternal("prediction processing failed", "model error", nil)
		prediction := &models.Prediction{
			ID:     predictionID,
			UserID: testdata.User1.ID,
			Status: models.PredictionFailedStatus,
			Result: nil,
			Error:  predictionError.Error(),
		}

		storeMock.EXPECT().
			GetPrediction(mock.Anything, predictionID).
			Return(prediction, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions/"+predictionID.String(), nil)
		req = mux.SetURLVars(req, map[string]string{predictionIDTag: predictionID.String()})

		rr := httptest.NewRecorder()
		server.getPrediction(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response models.Prediction
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, predictionID, response.ID)
		assert.Equal(t, models.PredictionFailedStatus, response.Status)
		assert.Nil(t, response.Result)
		assert.NotNil(t, response.Error)
	})

	t.Run("invalid prediction ID", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions/invalid-uuid", nil)
		req = mux.SetURLVars(req, map[string]string{predictionIDTag: "invalid-uuid"})

		rr := httptest.NewRecorder()
		server.getPrediction(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var errResp errlocal.BaseError
		err := json.NewDecoder(rr.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Message(), "invalid prediction ID")
	})

	t.Run("prediction not found", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		predictionID := uuid.New()

		storeMock.EXPECT().
			GetPrediction(mock.Anything, predictionID).
			Return(nil, errlocal.NewErrNotFound("prediction not found", "", nil)).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions/"+predictionID.String(), nil)
		req = mux.SetURLVars(req, map[string]string{predictionIDTag: predictionID.String()})

		rr := httptest.NewRecorder()
		server.getPrediction(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("store error", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		predictionID := uuid.New()

		storeMock.EXPECT().
			GetPrediction(mock.Anything, predictionID).
			Return(nil, errlocal.NewErrInternal("database error", "", nil)).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions/"+predictionID.String(), nil)
		req = mux.SetURLVars(req, map[string]string{predictionIDTag: predictionID.String()})

		rr := httptest.NewRecorder()
		server.getPrediction(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestListPredictions(t *testing.T) {
	t.Run("success with default pagination", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		user := testdata.User1
		predictions := []*models.Prediction{
			{
				ID:     uuid.New(),
				UserID: user.ID,
				Status: "completed",
			},
			{
				ID:     uuid.New(),
				UserID: user.ID,
				Status: "processing",
			},
		}

		storeMock.EXPECT().
			GetPredictionsByUserID(mock.Anything, user.ID, defaultOffset, defaultLimit).
			Return(predictions, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions", nil)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.listPredictions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response []*models.Prediction
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, predictions[0].ID, response[0].ID)
		assert.Equal(t, predictions[1].ID, response[1].ID)
	})

	t.Run("success with custom pagination", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		user := testdata.User1
		predictions := []*models.Prediction{
			{
				ID:     uuid.New(),
				UserID: user.ID,
				Status: "completed",
			},
		}

		storeMock.EXPECT().
			GetPredictionsByUserID(mock.Anything, user.ID, 10, 5).
			Return(predictions, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions?limit=5&offset=10", nil)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.listPredictions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response []*models.Prediction
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
	})

	t.Run("empty list", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		user := testdata.User1

		storeMock.EXPECT().
			GetPredictionsByUserID(mock.Anything, user.ID, defaultOffset, defaultLimit).
			Return([]*models.Prediction{}, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions", nil)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.listPredictions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var response []*models.Prediction
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Empty(t, response)
	})

	t.Run("store error", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		user := testdata.User1

		storeMock.EXPECT().
			GetPredictionsByUserID(mock.Anything, user.ID, defaultOffset, defaultLimit).
			Return(nil, errlocal.NewErrInternal("database error", "", nil)).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions", nil)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.listPredictions(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("invalid pagination parameters use defaults", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		user := testdata.User1
		predictions := []*models.Prediction{}

		storeMock.EXPECT().
			GetPredictionsByUserID(mock.Anything, user.ID, defaultOffset, defaultLimit).
			Return(predictions, nil).
			Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/predictions?limit=invalid&offset=bad", nil)
		req = req.WithContext(utils.SetUser(req.Context(), &user))

		rr := httptest.NewRecorder()
		server.listPredictions(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}
