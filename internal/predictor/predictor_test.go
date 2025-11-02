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
