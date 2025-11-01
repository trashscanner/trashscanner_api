package predictor

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type predictRequester interface {
	RequestPredict(ctx context.Context, scanURL string, predictionID uuid.UUID,
		optHeaders ...http.Header) (*predictResponse, error)
}

type Predictor struct {
	mu                sync.RWMutex
	log               *logging.Logger
	client            predictRequester
	store             store.Store
	scansInProcessing map[string]struct{}
	limiter           atomic.Int32
	limitRate         int32
}

func NewPredictor(logger *logging.Logger, store store.Store, cfg config.PredictorConfig) *Predictor {
	return &Predictor{
		log:               logger.WithPredictorTag(),
		store:             store,
		scansInProcessing: make(map[string]struct{}, cfg.MaxPredictionsInProcessing),
		client:            newPredictorClient(cfg, logger),
		limiter:           atomic.Int32{},
		limitRate:         int32(cfg.MaxPredictionsInProcessing),
	}
}

func (pr *Predictor) Predict(ctx context.Context, scanURL string) (*models.Prediction, error) {
	if pr.limiter.Load() >= pr.limitRate {
		return nil, errlocal.NewErrToManyRequests("to many predictions in processing")
	}
	pr.limiter.Add(1)

	if !pr.tryPutScanInProcessing(scanURL) {
		pr.limiter.Add(-1)
		return nil, errlocal.NewErrConflict("scan already in processing", "",
			map[string]any{"scan": scanURL})
	}
	user := utils.GetUser(ctx)

	pr.log.WithContext(ctx).Debugf("scan %s start processing", scanURL)
	prediction, err := pr.store.StartPrediction(ctx, user.ID, scanURL)
	if err != nil {
		pr.limiter.Add(-1)
		pr.deleteScanFromProcessing(scanURL)
		return nil, err
	}

	go pr.processPrediction(ctx, scanURL, prediction)

	return prediction, nil
}

func (pr *Predictor) processPrediction(ctx context.Context, scanURL string, prediction *models.Prediction) {
	defer func() { pr.limiter.Add(-1); pr.deleteScanFromProcessing(scanURL) }()
	logger := pr.log.WithContext(ctx)

	optsHeader := http.Header{}
	if requestID, ok := utils.GetRequestID(ctx); ok {
		optsHeader.Add("X-Request-ID", requestID)
	}

	var result any
	resp, reqErr := pr.client.RequestPredict(ctx, scanURL, prediction.ID, optsHeader)
	if reqErr != nil {
		logger.Errorf("error while process prediction %s: %v", prediction.ID.String(), reqErr)
		result = reqErr
	} else {
		logger.Debugf("result of process prediction %s: %v", prediction.ID.String(),
			models.NewPredictionResult(resp.Probabilities))
		result = models.NewPredictionResult(resp.Result)
	}

	if storeErr := pr.store.CompletePrediction(ctx, prediction.ID, result); storeErr != nil {
		logger.Errorf("error while complete prediction: %v", storeErr)
	}
}

func (pr *Predictor) tryPutScanInProcessing(scanURL string) bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	if _, ok := pr.scansInProcessing[scanURL]; ok {
		return false
	}
	pr.scansInProcessing[scanURL] = struct{}{}

	return true
}

func (pr *Predictor) deleteScanFromProcessing(scanURL string) {
	pr.mu.Lock()
	delete(pr.scansInProcessing, scanURL)
	pr.mu.Unlock()
}
