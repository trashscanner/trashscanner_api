package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	testdata "github.com/trashscanner/trashscanner_api/internal/testdata"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestLoginMiddleware(t *testing.T) {
	t.Run("invalid body", func(t *testing.T) {
		server, _, _, _ := newTestServer(t)
		body := loadJSONFixture(t, "login_empty_fields.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
		rr := httptest.NewRecorder()

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})

		server.loginMiddleware(next).ServeHTTP(rr, req)

		assert.False(t, called)
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "invalid request body", resp.Message())
	})

	t.Run("user not found", func(t *testing.T) {
		server, storeMock, _, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))

		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		storeMock.EXPECT().
			GetUserByLogin(mock.Anything, authReq.Login).
			Return((*models.User)(nil), errlocal.NewErrNotFound("user not found", "", nil))

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			storedReq := utils.GetRequestBody(r.Context()).(*dto.AuthRequest)
			assert.Equal(t, &authReq, storedReq)

			user := utils.GetUser(r.Context()).(models.User)
			assert.Equal(t, authReq.Login, user.Login)
			assert.True(t, user.ID == uuid.Nil)
			assert.NotEmpty(t, user.HashedPassword)
		})

		rr := httptest.NewRecorder()
		server.loginMiddleware(next).ServeHTTP(rr, req)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("user exists", func(t *testing.T) {
		server, storeMock, _, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))

		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		existing := testdata.User1
		storeMock.EXPECT().
			GetUserByLogin(mock.Anything, authReq.Login).
			Return(&existing, nil)

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			storedReq := utils.GetRequestBody(r.Context()).(*dto.AuthRequest)
			assert.Equal(t, &authReq, storedReq)

			user := utils.GetUser(r.Context()).(models.User)
			assert.Equal(t, existing.ID, user.ID)
			assert.Equal(t, existing.Login, user.Login)
		})

		rr := httptest.NewRecorder()
		server.loginMiddleware(next).ServeHTTP(rr, req)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("store error", func(t *testing.T) {
		server, storeMock, _, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))

		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		storeMock.EXPECT().
			GetUserByLogin(mock.Anything, authReq.Login).
			Return((*models.User)(nil), errors.New("db failure"))

		rr := httptest.NewRecorder()
		server.loginMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "failed check user", resp.Message())
	})
}

func TestLoginHandler(t *testing.T) {
	t.Run("creates new user", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body))
		req.Header.Set("X-Real-IP", testdata.TestIPAddress.String())
		req.Header.Set("X-Location", testdata.TestLocation)
		req.Header.Set("User-Agent", testdata.TestUserAgent)
		req.RemoteAddr = "198.51.100.10:12345"

		ctx := utils.SetUser(req.Context(), authReq.ToModel())
		ctx = utils.SetRequestBody(ctx, &authReq)
		req = req.WithContext(ctx)

		createdID := uuid.New()

		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool {
				assert.Equal(t, authReq.Login, u.Login)
				return true
			})).
			Run(func(_ context.Context, u *models.User) {
				u.ID = createdID
			}).
			Return(nil)

		authMock.EXPECT().
			CreateNewPair(mock.Anything, mock.MatchedBy(func(u models.User) bool {
				return u.ID == createdID && u.Login == authReq.Login
			})).
			Return(&auth.TokenPair{Access: "access", Refresh: "refresh"}, nil)

		rr := httptest.NewRecorder()
		server.login(rr, req)
		storeMock.AssertNotCalled(t, "InsertLoginHistory", mock.Anything, mock.Anything)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp dto.AuthResponse
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, createdID.String(), resp.User.ID)
		assert.Equal(t, authReq.Login, resp.User.Login)
		assert.Equal(t, "access", resp.Tokens.AccessToken)
		assert.Equal(t, "refresh", resp.Tokens.RefreshToken)
	})

	t.Run("existing user success", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		hashed, err := utils.HashPass(authReq.Password)
		require.NoError(t, err)

		existing := models.User{ID: testdata.User1ID, Login: authReq.Login, HashedPassword: hashed}
		ctx := utils.SetUser(context.Background(), existing)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body)).WithContext(ctx)
		req.Header.Set("X-Forwarded-For", testdata.TestIPAddress.String()+",192.0.2.1")
		req.Header.Set("User-Agent", testdata.TestUserAgent)
		req.RemoteAddr = "203.0.113.55:4567"

		authMock.EXPECT().
			CreateNewPair(mock.Anything, existing).
			Return(&auth.TokenPair{Access: "access", Refresh: "refresh"}, nil)

		storeMock.EXPECT().
			InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
				assert.Equal(t, existing.ID, history.UserID)
				assert.True(t, history.Success)
				require.NotNil(t, history.IpAddress)
				assert.Equal(t, testdata.TestIPAddress, *history.IpAddress)
				require.NotNil(t, history.UserAgent)
				assert.Equal(t, testdata.TestUserAgent, *history.UserAgent)
				return true
			})).
			Return(nil)

		rr := httptest.NewRecorder()
		server.login(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp dto.AuthResponse
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, existing.ID.String(), resp.User.ID)
	})

	t.Run("invalid password", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_invalid_credentials.json")
		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		hashed, err := utils.HashPass("correctpassword123")
		require.NoError(t, err)

		existing := models.User{ID: testdata.User1ID, Login: authReq.Login, HashedPassword: hashed}
		ctx := utils.SetUser(context.Background(), existing)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body)).WithContext(ctx)
		req.Header.Set("User-Agent", testdata.TestUserAgent)
		req.Header.Set("X-Location", testdata.TestLocation)

		storeMock.EXPECT().
			InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
				assert.Equal(t, existing.ID, history.UserID)
				assert.False(t, history.Success)
				if assert.NotNil(t, history.FailureReason) {
					assert.Contains(t, *history.FailureReason, "invalid credentials")
				}
				return true
			})).
			Return(nil)

		rr := httptest.NewRecorder()
		server.login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		authMock.AssertNotCalled(t, "CreateNewPair", mock.Anything, mock.Anything)
	})

	t.Run("create user error", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		ctx := utils.SetUser(context.Background(), authReq.ToModel())
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body)).WithContext(ctx)

		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.AnythingOfType("*models.User")).
			Return(errors.New("insert failed"))

		rr := httptest.NewRecorder()
		server.login(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		storeMock.AssertNotCalled(t, "InsertLoginHistory", mock.Anything, mock.Anything)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "failed to create user", resp.Message())
		authMock.AssertNotCalled(t, "CreateNewPair", mock.Anything, mock.Anything)
	})

	t.Run("token creation error", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.AuthRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		hashed, err := utils.HashPass(authReq.Password)
		require.NoError(t, err)

		existing := models.User{ID: testdata.User1ID, Login: authReq.Login, HashedPassword: hashed}
		ctx := utils.SetUser(context.Background(), existing)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewReader(body)).WithContext(ctx)

		authMock.EXPECT().
			CreateNewPair(mock.Anything, existing).
			Return((*auth.TokenPair)(nil), errors.New("sign failed"))

		storeMock.EXPECT().
			InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
				assert.False(t, history.Success)
				if assert.NotNil(t, history.FailureReason) {
					assert.Contains(t, *history.FailureReason, "failed to create tokens")
				}
				return true
			})).
			Return(nil)

		rr := httptest.NewRecorder()
		server.login(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestRefreshHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "refresh_valid.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewReader(body))

		var refreshReq dto.RefreshRequest
		require.NoError(t, json.Unmarshal(body, &refreshReq))

		authMock.EXPECT().
			Refresh(mock.Anything, refreshReq.RefreshToken).
			Return(&auth.TokenPair{Access: "access", Refresh: "refresh"}, nil)

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp dto.RefreshResponse
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "access", resp.Tokens.AccessToken)
		assert.Equal(t, "refresh", resp.Tokens.RefreshToken)
	})

	t.Run("invalid body", func(t *testing.T) {
		server, _, _, _ := newTestServer(t)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewReader([]byte("{")))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("token not found", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "refresh_invalid.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewReader(body))

		var refreshReq dto.RefreshRequest
		require.NoError(t, json.Unmarshal(body, &refreshReq))

		authMock.EXPECT().
			Refresh(mock.Anything, refreshReq.RefreshToken).
			Return((*auth.TokenPair)(nil), errlocal.NewErrNotFound("not found", "", nil))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("token expired", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "refresh_valid.json")
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewReader(body))

		var refreshReq dto.RefreshRequest
		require.NoError(t, json.Unmarshal(body, &refreshReq))

		authMock.EXPECT().
			Refresh(mock.Anything, refreshReq.RefreshToken).
			Return((*auth.TokenPair)(nil), jwt.ErrTokenExpired)

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("token revoked", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshReq := dto.RefreshRequest{RefreshToken: "revoked.token"}
		body, err := json.Marshal(refreshReq)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewReader(body))

		authMock.EXPECT().
			Refresh(mock.Anything, refreshReq.RefreshToken).
			Return((*auth.TokenPair)(nil), errors.New("token revoked"))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshReq := dto.RefreshRequest{RefreshToken: "broken.token"}
		body, err := json.Marshal(refreshReq)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", bytes.NewReader(body))

		authMock.EXPECT().
			Refresh(mock.Anything, refreshReq.RefreshToken).
			Return((*auth.TokenPair)(nil), errors.New("db down"))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
