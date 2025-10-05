package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	testdata "github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestInitRouter_NotFound(t *testing.T) {
	server, _, _ := newTestServer(t)
	server.initRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "endpoint not found")
}

func TestInitRouter_MethodNotAllowed(t *testing.T) {
	server, _, _ := newTestServer(t)
	server.initRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/login", nil)
	rr := httptest.NewRecorder()

	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
	assert.Contains(t, rr.Body.String(), "method not allowed")
}

func TestInitRouter_LoginFlow(t *testing.T) {
	server, storeMock, authMock := newTestServer(t)
	server.initRouter()

	body := loadJSONFixture(t, "login_valid.json")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-IP", testdata.TestIPAddress.String())
	req.Header.Set("User-Agent", testdata.TestUserAgent)

	var authReq dto.AuthRequest
	require.NoError(t, json.Unmarshal(body, &authReq))

	storeMock.EXPECT().
		GetUserByLogin(mock.Anything, authReq.Login).
		Return((*models.User)(nil), errlocal.NewErrNotFound("user not found", "", nil))

	createdID := uuid.New()
	storeMock.EXPECT().
		CreateUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool { return true })).
		Run(func(_ context.Context, u *models.User) {
			u.ID = createdID
		}).
		Return(nil)

	authMock.EXPECT().
		CreateNewPair(mock.Anything, mock.MatchedBy(func(u models.User) bool {
			return u.ID == createdID
		})).
		Return(&auth.TokenPair{Access: "access", Refresh: "refresh"}, nil)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	storeMock.AssertNotCalled(t, "InsertLoginHistory", mock.Anything, mock.Anything)

	var resp dto.AuthResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, createdID.String(), resp.User.ID)
}

func TestInitRouter_GetUserFlow(t *testing.T) {
	server, storeMock, authMock := newTestServer(t)
	server.initRouter()

	user := testdata.User1
	token := "access.token"
	claims := &auth.Claims{UserID: user.ID.String(), Login: user.Login}

	authMock.EXPECT().
		Parse(token).
		Return(claims, nil)

	storeMock.EXPECT().
		GetUser(mock.Anything, user.ID, true).
		Return(&user, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	server.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp models.User
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, user.ID, resp.ID)
}
