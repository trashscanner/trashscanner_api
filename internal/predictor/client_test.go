package predictor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/trashscanner/trashscanner_api/internal/config"
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
		logger: logging.NewLogger(config.Config{}),
		c:      http.DefaultClient,
		token:  "test-token",
	}
}

func (s *predictorClientTestSuite) TearDownTest() {
	if s.testServer != nil {
		s.testServer.Close()
	}
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
		s.setupTestServer(testSuccessPredictResponse, http.StatusOK)
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

	s.Run("error bad request", func() {
		s.setupTestServer(testErrorResponse, http.StatusBadRequest)

		res, err := s.client.RequestPredict(ctx, url, testdata.PredictionID)
		s.Nil(res)

		var badRequestErr *errlocal.ErrBadRequest
		s.ErrorAs(err, &badRequestErr)
	})

	s.Run("error forbidden", func() {
		s.setupTestServer(testErrorResponse, http.StatusForbidden)

		res, err := s.client.RequestPredict(ctx, url, testdata.PredictionID)
		s.Nil(res)

		var forbiddenErr *errlocal.ErrForbidden
		s.ErrorAs(err, &forbiddenErr)
	})

	s.Run("error internal server error", func() {
		s.setupTestServer(testErrorResponse, http.StatusInternalServerError)

		res, err := s.client.RequestPredict(ctx, url, testdata.PredictionID)
		s.Nil(res)

		var internalErr *errlocal.ErrInternal
		s.ErrorAs(err, &internalErr)
	})

	s.Run("with custom headers", func() {
		s.testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.Equal("test-token", r.Header.Get(tokenKey))
			s.Equal("test-request-id", r.Header.Get("X-Request-ID"))
			s.Equal("test-value", r.Header.Get("X-Custom-Header"))

			w.WriteHeader(http.StatusOK)
			w.Write(testSuccessPredictResponse)
		}))
		s.client.host = s.testServer.URL

		headers := http.Header{}
		headers.Add("X-Request-ID", "test-request-id")
		headers.Add("X-Custom-Header", "test-value")

		res, err := s.client.RequestPredict(ctx, url, testdata.PredictionID, headers)
		s.NoError(err)
		s.NotNil(res)
	})
}

func (s *predictorClientTestSuite) TestNewPredictorClient() {
	logger := logging.NewLogger(config.Config{})
	cfg := config.PredictorConfig{
		Address: "http://test-host",
		Token:   "test-token",
	}

	client := newPredictorClient(cfg, logger)

	s.NotNil(client)
	s.Equal("http://test-host", client.host)
	s.Equal("test-token", client.token)
	s.NotNil(client.c)
	s.Equal(predictorClientRequestTimeout, client.c.Timeout)
	s.NotNil(client.logger)
}

func (s *predictorClientTestSuite) TestPredictRequestBodyReader() {
	body := &predictRequestBody{
		ScanURL:      "test/scan/url",
		PredictionID: testdata.PredictionID.String(),
	}

	reader := body.Reader()
	s.NotNil(reader)

	var decoded predictRequestBody
	err := json.NewDecoder(reader).Decode(&decoded)
	s.NoError(err)
	s.Equal(body.ScanURL, decoded.ScanURL)
	s.Equal(body.PredictionID, decoded.PredictionID)
}
