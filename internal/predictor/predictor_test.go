package predictor

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store/mocks"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type predictorTestSuite struct {
	suite.Suite
	ctx       context.Context
	predictor *Predictor
	mClient   *mockPredictRequester
	mStore    *mocks.Store
}

func TestPredictorSuite(t *testing.T) {
	suite.Run(t, new(predictorTestSuite))
}

func (s *predictorTestSuite) SetupTest() {
	s.ctx = utils.SetUser(context.Background(), &testdata.User1)
	s.predictor = &Predictor{
		log:               logging.NewLogger(config.Config{}),
		limiter:           atomic.Int32{},
		limitRate:         10,
		scansInProcessing: make(map[string]struct{}),
	}

	s.mClient = newMockPredictRequester(s.T())
	s.mStore = mocks.NewStore(s.T())
	s.predictor.client = s.mClient
	s.predictor.store = s.mStore
}

func (s *predictorTestSuite) TestSuccessPredict() {
	testPrediction := testdata.NewPrediction
	predictionID := uuid.New()

	s.mStore.EXPECT().
		StartPrediction(mock.Anything, testPrediction.UserID, testPrediction.TrashScan).
		Run(func(_ context.Context, _ uuid.UUID, _ string) {
			testPrediction.ID = predictionID
		}).Return(&testPrediction, nil).Once()

	s.mClient.EXPECT().
		RequestPredict(mock.Anything, testdata.ScanURL, predictionID, mock.Anything).
		Return(&predictResponse{
			ID:            predictionID,
			Scan:          testPrediction.TrashScan,
			Result:        map[uint8]float64{1: 0.9},
			Probabilities: map[uint8]float64{1: 0.9, 2: 0.1, 3: 0.0},
		}, nil).Once()

	s.mStore.EXPECT().ExecTx(mock.Anything, mock.Anything).Return(nil).Once()

	result, err := s.predictor.Predict(s.ctx, testdata.ScanURL)
	s.NoError(err)

	s.Equal(&testPrediction, result)

	time.Sleep(time.Second)
}

func (s *predictorTestSuite) TestTooManyRequests() {
	for i := 0; i < 10; i++ {
		scanURL := testdata.User1ID.String() + "/scans/" + uuid.NewString()
		prediction := uuid.New()
		s.mStore.EXPECT().
			StartPrediction(mock.Anything, testdata.User1.ID, scanURL).
			Return(&models.Prediction{ID: prediction, TrashScan: scanURL}, nil).Once()
		s.mClient.EXPECT().RequestPredict(mock.Anything, scanURL, prediction, mock.Anything).
			Run(func(ctx context.Context, scanURL string, predictionID uuid.UUID, optHeaders ...http.Header) {
				time.Sleep(time.Second)
			}).
			Return(&predictResponse{
				ID:            prediction,
				Scan:          scanURL,
				Result:        map[uint8]float64{1: 0.9},
				Probabilities: map[uint8]float64{1: 0.9, 2: 0.1, 3: 0.0},
			}, nil).Once()
		s.mStore.EXPECT().ExecTx(mock.Anything, mock.Anything).Return(nil).Once()

		res, err := s.predictor.Predict(s.ctx, scanURL)
		s.NoError(err)
		s.Equal(res.ID, prediction)
		s.Equal(res.TrashScan, scanURL)
	}

	_, err := s.predictor.Predict(s.ctx, uuid.NewString())
	var tooManyReqErr *errlocal.ErrToManyRequests
	s.ErrorAs(err, &tooManyReqErr)

	time.Sleep(time.Second * 2)
}

func (s *predictorTestSuite) TestAlreadyProcessing() {
	scanURL := testdata.User1ID.String() + "/scans/" + uuid.NewString()
	prediction := uuid.New()

	s.mStore.EXPECT().
		StartPrediction(mock.Anything, testdata.User1.ID, scanURL).
		Return(&models.Prediction{ID: prediction, TrashScan: scanURL}, nil).Once()
	s.mClient.EXPECT().RequestPredict(mock.Anything, scanURL, prediction, mock.Anything).
		Run(func(ctx context.Context, scanURL string, predictionID uuid.UUID, optHeaders ...http.Header) {
			time.Sleep(time.Millisecond * 100)
		}).
		Return(&predictResponse{
			ID:            prediction,
			Scan:          scanURL,
			Result:        map[uint8]float64{1: 0.9},
			Probabilities: map[uint8]float64{1: 0.9, 2: 0.1, 3: 0.0},
		}, nil).Once()

	s.mStore.EXPECT().ExecTx(mock.Anything, mock.Anything).Return(nil).Once()

	res, err := s.predictor.Predict(s.ctx, scanURL)
	s.NoError(err)
	s.Equal(res.ID, prediction)
	s.Equal(res.TrashScan, scanURL)
	_, err = s.predictor.Predict(s.ctx, scanURL)

	var tooManyReqErr *errlocal.ErrConflict
	s.ErrorAs(err, &tooManyReqErr)

	time.Sleep(time.Second)
}

func (s *predictorTestSuite) TestPredict_StartPredictionError() {
	scanURL := testdata.User1ID.String() + "/scans/" + uuid.NewString()

	s.mStore.EXPECT().
		StartPrediction(mock.Anything, testdata.User1.ID, scanURL).
		Return(nil, errlocal.NewErrInternal("db error", "", nil)).Once()

	result, err := s.predictor.Predict(s.ctx, scanURL)
	s.Error(err)
	s.Nil(result)
	s.Equal(int32(0), s.predictor.limiter.Load())
}

func (s *predictorTestSuite) TestPredict_RequestPredictError() {
	testPrediction := testdata.NewPrediction
	predictionID := uuid.New()

	s.mStore.EXPECT().
		StartPrediction(mock.Anything, testPrediction.UserID, testPrediction.TrashScan).
		Run(func(_ context.Context, _ uuid.UUID, _ string) {
			testPrediction.ID = predictionID
		}).Return(&testPrediction, nil).Once()

	s.mClient.EXPECT().
		RequestPredict(mock.Anything, testdata.ScanURL, predictionID, mock.Anything).
		Return(nil, errlocal.NewErrInternal("request failed", "", nil)).Once()

	s.mStore.EXPECT().ExecTx(mock.Anything, mock.Anything).Return(nil).Once()

	result, err := s.predictor.Predict(s.ctx, testdata.ScanURL)
	s.NoError(err)
	s.Equal(&testPrediction, result)

	time.Sleep(time.Second)
}

func (s *predictorTestSuite) TestPredict_CompletePredictionError() {
	testPrediction := testdata.NewPrediction
	predictionID := uuid.New()

	s.mStore.EXPECT().
		StartPrediction(mock.Anything, testPrediction.UserID, testPrediction.TrashScan).
		Run(func(_ context.Context, _ uuid.UUID, _ string) {
			testPrediction.ID = predictionID
		}).Return(&testPrediction, nil).Once()

	s.mClient.EXPECT().
		RequestPredict(mock.Anything, testdata.ScanURL, predictionID, mock.Anything).
		Return(&predictResponse{
			ID:            predictionID,
			Scan:          testPrediction.TrashScan,
			Result:        map[uint8]float64{1: 0.9},
			Probabilities: map[uint8]float64{1: 0.9, 2: 0.1, 3: 0.0},
		}, nil).Once()

	s.mStore.EXPECT().ExecTx(mock.Anything, mock.Anything).
		Return(errlocal.NewErrInternal("tx error", "", nil)).Once()
	s.mStore.EXPECT().
		CompletePrediction(mock.Anything, predictionID, models.PredictionResult(nil),
			errlocal.NewErrInternal("tx error", "", nil)).Return(nil).Once()

	result, err := s.predictor.Predict(s.ctx, testdata.ScanURL)
	s.NoError(err)
	s.Equal(&testPrediction, result)

	time.Sleep(time.Second)
}

func (s *predictorTestSuite) TestTryPutScanInProcessing() {
	scanURL := "test/scan/url"

	// Первый раз должно быть успешно
	ok := s.predictor.tryPutScanInProcessing(scanURL)
	s.True(ok)
	s.Contains(s.predictor.scansInProcessing, scanURL)

	// Второй раз с тем же URL должно вернуть false
	ok = s.predictor.tryPutScanInProcessing(scanURL)
	s.False(ok)

	// Другой URL должен быть успешным
	ok = s.predictor.tryPutScanInProcessing("test/scan/url2")
	s.True(ok)
	s.Len(s.predictor.scansInProcessing, 2)
}

func (s *predictorTestSuite) TestDeleteScanFromProcessing() {
	scanURL := "test/scan/url"
	s.predictor.scansInProcessing[scanURL] = struct{}{}

	s.predictor.deleteScanFromProcessing(scanURL)
	s.NotContains(s.predictor.scansInProcessing, scanURL)

	// Удаление несуществующего не должно приводить к ошибке
	s.predictor.deleteScanFromProcessing("nonexistent")
	s.Len(s.predictor.scansInProcessing, 0)
}

func (s *predictorTestSuite) TestNewPredictor() {
	logger := logging.NewLogger(config.Config{})
	store := mocks.NewStore(s.T())
	cfg := config.PredictorConfig{
		Address:                    "http://predictor.test",
		Token:                      "test-token",
		MaxPredictionsInProcessing: 5,
	}

	predictor := NewPredictor(logger, store, cfg)

	s.NotNil(predictor)
	s.NotNil(predictor.log)
	s.NotNil(predictor.client)
	s.NotNil(predictor.store)
	s.Equal(int32(5), predictor.limitRate)
	s.Equal(int32(0), predictor.limiter.Load())
	s.NotNil(predictor.scansInProcessing)
	s.Len(predictor.scansInProcessing, 0)
}
