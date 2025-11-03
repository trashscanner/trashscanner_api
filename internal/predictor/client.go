package predictor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/logging"
)

const (
	predictEndpoint               = "/predict"
	predictorClientRequestTimeout = time.Second * 10
	tokenKey                      = "token"
)

type predictorClient struct {
	logger *logging.Logger
	c      *http.Client
	host   string
	token  string
}

func newPredictorClient(cfg config.PredictorConfig, log *logging.Logger) *predictorClient {
	return &predictorClient{
		c:      &http.Client{Timeout: predictorClientRequestTimeout},
		host:   cfg.Address,
		token:  cfg.Token,
		logger: log.WithPredictorClientTag(),
	}
}

type predictResponse struct {
	ID            uuid.UUID         `json:"prediction_id"`
	Scan          string            `json:"target"`
	Result        map[uint8]float64 `json:"result"`
	Probabilities map[uint8]float64 `json:"probabilities"`
}

type errorResponse struct {
	Detail string `json:"detail"`
}

type predictRequestBody struct {
	ScanURL      string `json:"scan_url"`
	PredictionID string `json:"prediction_id"`
}

//nolint:errchkjson // do not check errors
func (b *predictRequestBody) Reader() io.Reader {
	jsonStr, _ := json.Marshal(b)

	return bytes.NewReader(jsonStr)
}

func (c *predictorClient) RequestPredict(
	ctx context.Context,
	scanURL string,
	predictionID uuid.UUID,
	optHeaders ...http.Header,
) (*predictResponse, error) {
	reqBody := predictRequestBody{
		ScanURL:      scanURL,
		PredictionID: predictionID.String(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+predictEndpoint, reqBody.Reader())
	if err != nil {
		return nil, err
	}

	for _, h := range optHeaders {
		for k, v := range h {
			req.Header[k] = v
		}
	}
	req.Header.Add(tokenKey, c.token)

	resp, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	decoder := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(decoder, resp.StatusCode)
	}

	body := new(predictResponse)

	return body, decoder.Decode(body)
}

func parseErrorResponse(decoder *json.Decoder, code int) error {
	var errResp errorResponse
	_ = decoder.Decode(&errResp)
	msg := "error while requesting prediction"

	switch code {
	case http.StatusBadRequest:
		return errlocal.NewErrBadRequest(msg, errResp.Detail, nil)
	case http.StatusForbidden:
		return errlocal.NewErrForbidden(msg, errResp.Detail, nil)
	case http.StatusNotFound:
		return errlocal.NewErrNotFound(msg, errResp.Detail, nil)
	default:
	}

	return errlocal.NewErrInternal(msg, errResp.Detail, nil)
}
