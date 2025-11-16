package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/auth/mocks"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	filestoremocks "github.com/trashscanner/trashscanner_api/internal/filestore/mocks"
	"github.com/trashscanner/trashscanner_api/internal/logging"
	storemocks "github.com/trashscanner/trashscanner_api/internal/store/mocks"
)

func TestNewServer(t *testing.T) {
	cfg := config.Config{
		Server: config.ServerConfig{Host: "127.0.0.1", Port: "9090"},
		Log:    config.LogConfig{Level: "info", Format: "text"},
	}
	store := storemocks.NewStore(t)
	fileStore := filestoremocks.NewFileStore(t)
	authManager := mocks.NewAuthManager(t)
	predictor := newMockPredictor(t)
	logger := logging.NewLogger(cfg)

	server := NewServer(cfg, store, fileStore, authManager, predictor, logger)

	require.NotNil(t, server.router)
	require.NotNil(t, server.s)
	assert.Equal(t, "127.0.0.1:9090", server.s.Addr)
	assert.Equal(t, defaultTimeout, server.s.WriteTimeout)
	assert.Equal(t, defaultTimeout, server.s.ReadTimeout)
	assert.Equal(t, store, server.store)
	assert.Equal(t, authManager, server.authManager)
}

func TestWriteResponse(t *testing.T) {
	t.Run("with data", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		data := map[string]any{"message": "ok"}

		server.WriteResponse(rr, req, http.StatusAccepted, data)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"message": "ok"}`, rr.Body.String())
	})

	t.Run("with nil data and non-204 status", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		server.WriteResponse(rr, req, http.StatusOK, nil)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"status": "OK"}`, rr.Body.String())
	})

	t.Run("with nil data and 204 status", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/test", nil)

		server.WriteResponse(rr, req, http.StatusNoContent, nil)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.JSONEq(t, `null`, rr.Body.String())
	})

	t.Run("with unencodable data", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		invalidData := make(chan int)

		server.WriteResponse(rr, req, http.StatusOK, invalidData)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), "failed to encode response")
	})
}

func TestWriteError(t *testing.T) {
	t.Run("with LocalError", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		err := errlocal.NewErrInternal("boom", errors.New("boom").Error(), nil)

		server.WriteError(rr, req, err)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), "boom")
	})

	t.Run("with regular error", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		err := errors.New("regular error")

		server.WriteError(rr, req, err)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("with BadRequest error", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		err := errlocal.NewErrBadRequest("invalid input", "field is required", nil)

		server.WriteError(rr, req, err)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), "invalid input")
	})

	t.Run("with NotFound error", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		err := errlocal.NewErrNotFound("not found", "resource not found", nil)

		server.WriteError(rr, req, err)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
		assert.Contains(t, rr.Body.String(), "not found")
	})
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		healthy        bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "healthy server",
			healthy:        true,
			expectedStatus: http.StatusOK,
			expectedBody:   "true",
		},
		{
			name:           "unhealthy server",
			healthy:        false,
			expectedStatus: http.StatusOK,
			expectedBody:   "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _, _, _, _ := newTestServer(t)
			server.healthy = tt.healthy

			rr := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/health", nil)

			server.healthCheck(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
			assert.JSONEq(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestShutdownWithoutStart(t *testing.T) {
	server, _, _, _, _ := newTestServer(t)

	start := time.Now()
	err := server.Shutdown()
	elapsed := time.Since(start)

	assert.LessOrEqual(t, elapsed, 100*time.Millisecond)
	assert.NoError(t, err)
}
