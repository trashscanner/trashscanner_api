package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

// StartPrediction godoc
// @Summary Start a new prediction
// @Description Start a new prediction for a user
// @Tags predictions
// @Accept multipart/form-data
// @Produce json
// @Param scan formData file true "File to upload"
// @Success 202 {object} dto.PredictionResponse "Prediction result"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /predictions [post]
func (s *Server) startPrediction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := utils.GetUser(ctx)

	file, err := dto.GetScanFromMultipartForm(r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("bad format of file", err.Error(), nil))
		return
	}
	file.ID = uuid.New()

	fileURL, err := s.fileStore.UploadScan(ctx, user.ID.String(), file)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrInternal("failed to upload scan", err.Error(), nil))
		return
	}

	newPrediction, err := s.predictor.Predict(ctx, fileURL)
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusAccepted, newPrediction)
}

// StartPrediction godoc
// @Summary Get a prediction
// @Description Get a prediction by ID
// @Tags predictions
// @Accept json
// @Produce json
// @Param PredictionID path string true "Prediction ID UUID format"
// @Success 200 {object} dto.PredictionResponse "Prediction result"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "Resource not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /predictions/{PredictionID} [get]
func (s *Server) getPrediction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	predictionID, err := uuid.Parse(mux.Vars(r)[predictionIDTag])
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid prediction ID", err.Error(), nil))
		return
	}

	prediction, err := s.store.GetPrediction(ctx, predictionID)
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusOK, prediction)
}

// StartPrediction godoc
// @Summary Get a list of predictions
// @Description Get a list of predictions for a user
// @Tags predictions
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default 100
// @Param offset query int false "Offset" default 0
// @Success 200 {object} []dto.PredictionResponse "Prediction result"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /predictions [get]
func (s *Server) listPredictions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := utils.GetUser(ctx)

	offset := utils.GetQueryParam(r, offsetQueryKey, defaultOffset)
	limit := utils.GetQueryParam(r, limitQueryKey, defaultLimit)

	predictions, err := s.store.GetPredictionsByUserID(ctx, user.ID, offset, limit)
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusOK, predictions)
}
