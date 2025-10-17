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

		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		storeMock.EXPECT().
			GetUserByLogin(mock.Anything, authReq.Login).
			Return((*models.User)(nil), errlocal.NewErrNotFound("user not found", "", nil))

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			storedReq := utils.GetRequestBody(r.Context()).(*dto.LoginUserRequest)
			assert.Equal(t, &authReq, storedReq)

			user := utils.GetUser(r.Context())
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

		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		existing := testdata.User1
		storeMock.EXPECT().
			GetUserByLogin(mock.Anything, authReq.Login).
			Return(&existing, nil)

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			storedReq := utils.GetRequestBody(r.Context()).(*dto.LoginUserRequest)
			assert.Equal(t, &authReq, storedReq)

			user := utils.GetUser(r.Context())
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
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))

		var authReq dto.LoginUserRequest
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
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))
		authReq.Name = "testname" // Добавляем Name для register

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
		req.Header.Set("X-Real-IP", testdata.TestIPAddress.String())
		req.Header.Set("X-Location", testdata.TestLocation)
		req.Header.Set("User-Agent", testdata.TestUserAgent)
		req.RemoteAddr = "198.51.100.10:12345"

		user := authReq.ToModel()
		ctx := utils.SetUser(req.Context(), &user)
		ctx = utils.SetRequestBody(ctx, &authReq)
		req = req.WithContext(ctx)

		createdID := uuid.New()

		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool {
				assert.Equal(t, authReq.Login, u.Login)
				assert.Equal(t, authReq.Name, u.Name)
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

		storeMock.EXPECT().
			InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
				assert.Equal(t, createdID, history.UserID)
				assert.True(t, history.Success)
				return true
			})).
			Return(nil)

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp dto.AuthResponse
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, createdID.String(), resp.User.ID)
		assert.Equal(t, authReq.Login, resp.User.Login)

		// Проверяем, что токены установлены в cookies
		cookies := rr.Result().Cookies()
		var accessCookie, refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "access_token" {
				accessCookie = c
			}
			if c.Name == "refresh_token" {
				refreshCookie = c
			}
		}
		require.NotNil(t, accessCookie, "access_token cookie should be set")
		require.NotNil(t, refreshCookie, "refresh_token cookie should be set")
		assert.Equal(t, "access", accessCookie.Value)
		assert.Equal(t, "refresh", refreshCookie.Value)
	})

	t.Run("existing user success", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		hashed, err := utils.HashPass(authReq.Password)
		require.NoError(t, err)

		existing := models.User{ID: testdata.User1ID, Login: authReq.Login, HashedPassword: hashed}
		ctx := utils.SetUser(context.Background(), &existing)
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
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		hashed, err := utils.HashPass("correctpassword123")
		require.NoError(t, err)

		existing := models.User{ID: testdata.User1ID, Login: authReq.Login, HashedPassword: hashed}
		ctx := utils.SetUser(context.Background(), &existing)
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
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		user := authReq.ToModel()
		user.Name = "testname" // Устанавливаем имя, чтобы пройти проверку
		ctx := utils.SetUser(context.Background(), &user)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body)).WithContext(ctx)

		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.AnythingOfType("*models.User")).
			Return(errlocal.NewErrInternal("insert failed", "", nil))

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		storeMock.AssertNotCalled(t, "InsertLoginHistory", mock.Anything, mock.Anything)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "insert failed", resp.Message())
		authMock.AssertNotCalled(t, "CreateNewPair", mock.Anything, mock.Anything)
	})

	t.Run("token creation error", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))

		hashed, err := utils.HashPass(authReq.Password)
		require.NoError(t, err)

		existing := models.User{ID: testdata.User1ID, Login: authReq.Login, HashedPassword: hashed}
		ctx := utils.SetUser(context.Background(), &existing)
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

func TestRegisterHandler(t *testing.T) {
	t.Run("success - creates new user with all fields", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))
		authReq.Name = "testname"

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body))
		req.Header.Set("X-Real-IP", testdata.TestIPAddress.String())
		req.Header.Set("X-Location", testdata.TestLocation)
		req.Header.Set("User-Agent", testdata.TestUserAgent)

		user := authReq.ToModel()
		ctx := utils.SetUser(req.Context(), &user)
		ctx = utils.SetRequestBody(ctx, &authReq)
		req = req.WithContext(ctx)

		createdID := uuid.New()

		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.MatchedBy(func(u *models.User) bool {
				assert.Equal(t, authReq.Login, u.Login)
				assert.Equal(t, authReq.Name, u.Name)
				assert.NotEmpty(t, u.HashedPassword)
				return true
			})).
			Run(func(_ context.Context, u *models.User) {
				u.ID = createdID
			}).
			Return(nil)

		authMock.EXPECT().
			CreateNewPair(mock.Anything, mock.MatchedBy(func(u models.User) bool {
				return u.ID == createdID && u.Login == authReq.Login && u.Name == authReq.Name
			})).
			Return(&auth.TokenPair{Access: "new_access_token", Refresh: "new_refresh_token"}, nil)

		storeMock.EXPECT().
			InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
				assert.Equal(t, createdID, history.UserID)
				assert.True(t, history.Success)
				assert.Nil(t, history.FailureReason)
				return true
			})).
			Return(nil)

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var resp dto.AuthResponse
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, createdID.String(), resp.User.ID)
		assert.Equal(t, authReq.Login, resp.User.Login)

		cookies := rr.Result().Cookies()
		var accessCookie, refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "access_token" {
				accessCookie = c
			}
			if c.Name == "refresh_token" {
				refreshCookie = c
			}
		}
		require.NotNil(t, accessCookie)
		require.NotNil(t, refreshCookie)
		assert.Equal(t, "new_access_token", accessCookie.Value)
		assert.Equal(t, "new_refresh_token", refreshCookie.Value)
		assert.True(t, accessCookie.HttpOnly)
		assert.True(t, refreshCookie.HttpOnly)
	})

	t.Run("error - user already exists", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))
		authReq.Name = "testname"

		existing := testdata.User1
		existing.Login = authReq.Login
		ctx := utils.SetUser(context.Background(), &existing)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body)).WithContext(ctx)

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Contains(t, resp.Message(), "already exists")
		authMock.AssertNotCalled(t, "CreateNewPair", mock.Anything, mock.Anything)
	})

	t.Run("error - name is required", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))
		// Не устанавливаем Name

		user := authReq.ToModel()
		ctx := utils.SetUser(context.Background(), &user)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body)).WithContext(ctx)

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "name is required", resp.Message())
		storeMock.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
		authMock.AssertNotCalled(t, "CreateNewPair", mock.Anything, mock.Anything)
	})

	t.Run("error - create user fails", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))
		authReq.Name = "testname"

		user := authReq.ToModel()
		ctx := utils.SetUser(context.Background(), &user)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body)).WithContext(ctx)

		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.AnythingOfType("*models.User")).
			Return(errlocal.NewErrInternal("database connection failed", "", nil))

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "database connection failed", resp.Message())
		authMock.AssertNotCalled(t, "CreateNewPair", mock.Anything, mock.Anything)
	})

	t.Run("error - token creation fails", func(t *testing.T) {
		server, storeMock, authMock, _ := newTestServer(t)

		body := loadJSONFixture(t, "login_valid.json")
		var authReq dto.LoginUserRequest
		require.NoError(t, json.Unmarshal(body, &authReq))
		authReq.Name = "testname"

		user := authReq.ToModel()
		ctx := utils.SetUser(context.Background(), &user)
		ctx = utils.SetRequestBody(ctx, &authReq)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewReader(body)).WithContext(ctx)
		req.Header.Set("User-Agent", testdata.TestUserAgent)

		createdID := uuid.New()
		storeMock.EXPECT().
			CreateUser(mock.Anything, mock.AnythingOfType("*models.User")).
			Run(func(_ context.Context, u *models.User) {
				u.ID = createdID
			}).
			Return(nil)

		authMock.EXPECT().
			CreateNewPair(mock.Anything, mock.AnythingOfType("models.User")).
			Return((*auth.TokenPair)(nil), errors.New("jwt signing failed"))

		storeMock.EXPECT().
			InsertLoginHistory(mock.Anything, mock.MatchedBy(func(history *models.LoginHistory) bool {
				assert.Equal(t, createdID, history.UserID)
				assert.False(t, history.Success)
				assert.NotNil(t, history.FailureReason)
				assert.Contains(t, *history.FailureReason, "jwt signing failed")
				return true
			})).
			Return(nil)

		rr := httptest.NewRecorder()
		server.register(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "error create tokens", resp.Message())
	})
}

func TestRefreshHandler(t *testing.T) {
	t.Run("success - refresh tokens via cookie", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshToken := "valid.refresh.token"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})

		authMock.EXPECT().
			Refresh(mock.Anything, refreshToken).
			Return(&auth.TokenPair{Access: "new_access", Refresh: "new_refresh"}, nil)

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)

		// Проверяем, что новые токены установлены в cookies
		cookies := rr.Result().Cookies()
		var accessCookie, refreshCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "access_token" {
				accessCookie = c
			}
			if c.Name == "refresh_token" {
				refreshCookie = c
			}
		}
		require.NotNil(t, accessCookie, "access_token cookie should be set")
		require.NotNil(t, refreshCookie, "refresh_token cookie should be set")
		assert.Equal(t, "new_access", accessCookie.Value)
		assert.Equal(t, "new_refresh", refreshCookie.Value)
		assert.True(t, accessCookie.HttpOnly)
		assert.True(t, refreshCookie.HttpOnly)
	})

	t.Run("error - missing refresh token cookie", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "missing refresh token cookie", resp.Message())
		authMock.AssertNotCalled(t, "Refresh", mock.Anything, mock.Anything)
	})

	t.Run("error - token not found", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshToken := "invalid.token"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})

		authMock.EXPECT().
			Refresh(mock.Anything, refreshToken).
			Return((*auth.TokenPair)(nil), errlocal.NewErrNotFound("token not found", "", nil))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "token not found", resp.Message())
	})

	t.Run("error - token expired", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshToken := "expired.token"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})

		authMock.EXPECT().
			Refresh(mock.Anything, refreshToken).
			Return((*auth.TokenPair)(nil), jwt.ErrTokenExpired)

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "token expired", resp.Message())
	})

	t.Run("error - internal server error", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshToken := "valid.token"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})

		authMock.EXPECT().
			Refresh(mock.Anything, refreshToken).
			Return((*auth.TokenPair)(nil), errors.New("database connection failed"))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		var resp errlocal.BaseError
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, "failed to refresh tokens", resp.Message())
	})

	t.Run("error - revoked token", func(t *testing.T) {
		server, _, authMock, _ := newTestServer(t)

		refreshToken := "revoked.token"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: refreshToken,
		})

		authMock.EXPECT().
			Refresh(mock.Anything, refreshToken).
			Return((*auth.TokenPair)(nil), errlocal.NewErrUnauthorized("token revoked", "", nil))

		rr := httptest.NewRecorder()
		server.refresh(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
