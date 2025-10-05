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
	storemocks "github.com/trashscanner/trashscanner_api/internal/store/mocks"
)

func TestNewServer(t *testing.T) {
	cfg := config.Config{Server: config.ServerConfig{Host: "127.0.0.1", Port: "9090"}}
	store := storemocks.NewStore(t)
	authManager := mocks.NewAuthManager(t)

	server := NewServer(cfg, store, authManager)

	require.NotNil(t, server.router)
	require.NotNil(t, server.s)
	assert.Equal(t, "127.0.0.1:9090", server.s.Addr)
	assert.Equal(t, defaultTimeout, server.s.WriteTimeout)
	assert.Equal(t, defaultTimeout, server.s.ReadTimeout)
	assert.Equal(t, store, server.store)
	assert.Equal(t, authManager, server.authManager)
}

func TestWriteResponse(t *testing.T) {
	server, _, _ := newTestServer(t)

	rr := httptest.NewRecorder()
	data := map[string]any{"message": "ok"}

	server.WriteResponse(rr, http.StatusAccepted, data)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"message": "ok"}`, rr.Body.String())
}

func TestWriteError(t *testing.T) {
	server, _, _ := newTestServer(t)

	rr := httptest.NewRecorder()
	err := errlocal.NewErrInternal("boom", errors.New("boom").Error(), nil)

	server.WriteError(rr, err)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.Contains(t, rr.Body.String(), "boom")
}

func TestShutdownWithoutStart(t *testing.T) {
	server, _, _ := newTestServer(t)

	start := time.Now()
	err := server.Shutdown()
	elapsed := time.Since(start)

	assert.LessOrEqual(t, elapsed, 100*time.Millisecond)
	assert.NoError(t, err)
}
