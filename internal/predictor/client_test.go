package predictor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

var (
	testSuccessPredictResponse = []byte(`
{
  "prediction_id": "00e0cc2f-47e0-4e0d-a0f5-401fe9f0f5d6",
  "target": "a34d8ee6-fe5f-4795-ba7e-6e127ec2aa02",
  "result": {
    "2": 0.9814258217811584
  },
  "probabilities": {
    "0": 0.010785728693008423,
    "1": 0.00405508279800415,
    "2": 0.9814258217811584,
    "3": 0.017564475536346436,
    "4": 0.038641273975372314,
    "5": 0.00656464695930481
  }
}`)
	testErrorResponse = []byte(`
{
	"detail": "Image not found",
}`)
)

type predictorClientTestSuite struct {
	suite.Suite
	client     *predictorClient
	testServer *httptest.Server
}

func (s *predictorClientTestSuite) SetupTest() {
	s.client = &predictorClient{
		logger: &logging.Logger{},
		c:      http.DefaultClient,
		token:  "test-token",
	}
}

func (s *predictorClientTestSuite) TearDownTest() {
	s.testServer.Close()
}

func TestPredictorClientTestSuite(t *testing.T) {
	suite.Run(t, new(predictorClientTestSuite))
}

func (s *predictorClientTestSuite) setupTestServer(
	expectedBody []byte,
	expectedCode int,
) {
	s.testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("test-token", r.Header.Get(tokenKey))

		var reqBody predictRequestBody
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		s.NoError(err)
		s.NotZero(reqBody)
		s.Equal(testdata.PredictionID.String(), reqBody.PredictionID)

		w.WriteHeader(expectedCode)
		w.Write(expectedBody)
	}))
	s.client.host = s.testServer.URL
}

func (s *predictorClientTestSuite) TestRequestPredict() {
	url := "test/user/img"
	ctx := context.Background()

	s.Run("success", func() {
		s.setupTestServer(testSuccessPredictResponse, http.StatusAccepted)
		expectedResult := &predictResponse{
			ID:     testdata.PredictionID,
			Scan:   testdata.ScanID.String(),
			Result: map[uint8]float64{2: 0.9814258217811584},
			Probabilities: map[uint8]float64{
				0: 0.010785728693008423,
				1: 0.00405508279800415,
				2: 0.9814258217811584,
				3: 0.017564475536346436,
				4: 0.038641273975372314,
				5: 0.00656464695930481,
			},
		}

		res, err := s.client.RequestPredict(ctx, url, testdata.PredictionID)
		s.NoError(err)
		s.Equal(expectedResult, res)
	})

	s.Run("error not found", func() {
		s.setupTestServer(testErrorResponse, http.StatusNotFound)

		res, err := s.client.RequestPredict(ctx, url, testdata.PredictionID)
		s.Nil(res)

		var notFoundErr *errlocal.ErrNotFound
		s.ErrorAs(err, &notFoundErr)
	})
}
