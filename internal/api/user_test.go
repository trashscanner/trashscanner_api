package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/auth"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	testdata "github.com/trashscanner/trashscanner_api/internal/testdata"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestAuthMiddleware(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, _, authMock := newTestServer(t)

		token := "access.token"
		claims := &auth.Claims{UserID: testdata.User1.ID.String(), Login: testdata.User1.Login}

		authMock.EXPECT().
			Parse(token).
			Return(claims, nil)

		nextCalled := false
		handler := server.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			user := utils.GetUser(r.Context()).(models.User)
			assert.Equal(t, claims.UserID, user.ID.String())
			assert.Equal(t, claims.Login, user.Login)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("missing header", func(t *testing.T) {
		server, _, _ := newTestServer(t)

		handler := server.authMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("should not be called")
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		server, _, authMock := newTestServer(t)

		token := "bad.token"
		authMock.EXPECT().
			Parse(token).
			Return((*auth.Claims)(nil), errors.New("invalid"))

		handler := server.authMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("should not be called")
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestUserMiddleware(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		user := testdata.User1
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user.ID.String(), nil)
		req = req.WithContext(utils.SetUser(req.Context(), models.User{ID: user.ID, Login: user.Login}))

		storeMock.EXPECT().
			GetUser(mock.Anything, user.ID, true).
			Return(&user, nil)

		nextCalled := false
		handler := server.userMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			stored := utils.GetUser(r.Context()).(models.User)
			assert.Equal(t, user.ID, stored.ID)
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+testdata.User1.ID.String(), nil)
		req = req.WithContext(utils.SetUser(req.Context(), models.User{ID: testdata.User1.ID, Login: testdata.User1.Login}))

		storeMock.EXPECT().
			GetUser(mock.Anything, testdata.User1.ID, true).
			Return((*models.User)(nil), errlocal.NewErrNotFound("user not found", "", nil))

		handler := server.userMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("should not be called")
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("internal error", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+testdata.User1.ID.String(), nil)
		req = req.WithContext(utils.SetUser(req.Context(), models.User{ID: testdata.User1.ID, Login: testdata.User1.Login}))

		storeMock.EXPECT().
			GetUser(mock.Anything, testdata.User1.ID, true).
			Return((*models.User)(nil), errors.New("db down"))

		handler := server.userMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("should not be called")
		}))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestGetUser(t *testing.T) {
	server, _, _ := newTestServer(t)

	user := testdata.User1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+user.ID.String(), nil)
	req = req.WithContext(utils.SetUser(req.Context(), user))

	rr := httptest.NewRecorder()
	server.getUser(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp models.User
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, user.ID, resp.ID)
}

func TestDeleteUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		user := testdata.User1
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
		req = req.WithContext(utils.SetUser(req.Context(), user))

		storeMock.EXPECT().
			DeleteUser(mock.Anything, user.ID).
			Return(nil)

		rr := httptest.NewRecorder()
		server.deleteUser(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("delete error", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		user := testdata.User1
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
		req = req.WithContext(utils.SetUser(req.Context(), user))

		storeMock.EXPECT().
			DeleteUser(mock.Anything, user.ID).
			Return(errlocal.NewErrInternal("db error", "", nil))

		rr := httptest.NewRecorder()
		server.deleteUser(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestSwitchPassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		oldPassword := "oldpass123"
		hashedOldPassword, _ := utils.HashPass(oldPassword)

		user := testdata.User1
		user.HashedPassword = hashedOldPassword

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/switch-password",
			loadJSONFixtureReader(t, "switch_password_valid.json"))
		req = req.WithContext(utils.SetUser(req.Context(), user))

		storeMock.EXPECT().
			UpdateUserPass(mock.Anything, user.ID, mock.AnythingOfType("string")).
			Return(nil)

		rr := httptest.NewRecorder()
		server.switchPassword(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)
	})

	t.Run("invalid body", func(t *testing.T) {
		server, _, _ := newTestServer(t)

		user := testdata.User1
		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/switch-password",
			loadJSONFixtureReader(t, "switch_password_empty.json"))
		req = req.WithContext(utils.SetUser(req.Context(), user))

		rr := httptest.NewRecorder()
		server.switchPassword(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("old password mismatch", func(t *testing.T) {
		server, _, _ := newTestServer(t)

		oldPassword := "correctpass"
		hashedOldPassword, _ := utils.HashPass(oldPassword)

		user := testdata.User1
		user.HashedPassword = hashedOldPassword

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/switch-password",
			loadJSONFixtureReader(t, "switch_password_wrong_old.json"))
		req = req.WithContext(utils.SetUser(req.Context(), user))

		rr := httptest.NewRecorder()
		server.switchPassword(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("update error", func(t *testing.T) {
		server, storeMock, _ := newTestServer(t)

		oldPassword := "oldpass123"
		hashedOldPassword, _ := utils.HashPass(oldPassword)

		user := testdata.User1
		user.HashedPassword = hashedOldPassword

		req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/switch-password",
			loadJSONFixtureReader(t, "switch_password_valid.json"))
		req = req.WithContext(utils.SetUser(req.Context(), user))

		storeMock.EXPECT().
			UpdateUserPass(mock.Anything, user.ID, mock.AnythingOfType("string")).
			Return(errlocal.NewErrInternal("db error", "", nil))

		rr := httptest.NewRecorder()
		server.switchPassword(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestLogout(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server, _, authMock := newTestServer(t)

		user := testdata.User1
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/logout", nil)
		req = req.WithContext(utils.SetUser(req.Context(), user))

		authMock.EXPECT().
			RevokeAllUserTokens(mock.Anything, user.ID).
			Return(nil)

		rr := httptest.NewRecorder()
		server.logout(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var resp models.User
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		assert.Equal(t, user.ID, resp.ID)
	})

	t.Run("revoke tokens error", func(t *testing.T) {
		server, _, authMock := newTestServer(t)

		user := testdata.User1
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/logout", nil)
		req = req.WithContext(utils.SetUser(req.Context(), user))

		authMock.EXPECT().
			RevokeAllUserTokens(mock.Anything, user.ID).
			Return(errors.New("db error"))

		rr := httptest.NewRecorder()
		server.logout(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
