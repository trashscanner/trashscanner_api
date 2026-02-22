package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

func TestGetUsersList(t *testing.T) {
	t.Run("success_admin_users", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=10&offset=0", nil)
		rr := httptest.NewRecorder()

		now := time.Now()
		userID := uuid.New()
		loginTime := now.Add(-time.Hour)

		dbUsers := []models.User{
			{
				ID:          userID,
				Login:       "user1",
				Name:        "User One",
				Role:        "user",
				CreatedAt:   now,
				UpdatedAt:   now,
				LastLoginAt: &loginTime,
				Stat: &models.Stat{
					Rating:        100,
					LastScannedAt: now,
				},
			},
		}

		storeMock.EXPECT().GetAdminUsers(mock.Anything, int32(10), int32(0)).Return(dbUsers, nil)
		storeMock.EXPECT().CountUsers(mock.Anything).Return(int64(1), nil)

		server.getUsersList(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var res dto.AdminUserListResponse
		err := json.NewDecoder(rr.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, int64(1), res.TotalCount)
		assert.Equal(t, 10, res.Limit)
		assert.Equal(t, 0, res.Offset)
		assert.Len(t, res.Users, 1)

		user := res.Users[0]
		assert.Equal(t, "user1", user.Login)
		require.NotNil(t, user.LastLoginAt)
		assert.Equal(t, loginTime.Unix(), user.LastLoginAt.Unix())
		require.NotNil(t, user.Stat)
		assert.Equal(t, 100, user.Stat.Rating)
		require.NotNil(t, user.Stat.LastScannedAt)
		assert.Equal(t, now.Unix(), user.Stat.LastScannedAt.Unix())
	})

	t.Run("invalid_pagination", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=-10&offset=-1", nil)
		rr := httptest.NewRecorder()

		// defaults to 0 and 100 if invalid
		storeMock.EXPECT().GetAdminUsers(mock.Anything, int32(100), int32(0)).Return([]models.User{}, nil)
		storeMock.EXPECT().CountUsers(mock.Anything).Return(int64(0), nil)

		server.getUsersList(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("error_get_admin_users", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=10&offset=0", nil)
		rr := httptest.NewRecorder()

		storeMock.EXPECT().GetAdminUsers(mock.Anything, int32(10), int32(0)).Return(nil, assert.AnError)

		server.getUsersList(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("error_count_users", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?limit=10&offset=0", nil)
		rr := httptest.NewRecorder()

		storeMock.EXPECT().GetAdminUsers(mock.Anything, int32(10), int32(0)).Return([]models.User{}, nil)
		storeMock.EXPECT().CountUsers(mock.Anything).Return(int64(0), assert.AnError)

		server.getUsersList(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}

func TestCreateUser_Admin(t *testing.T) {
	t.Run("success_create_admin", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		reqDto := dto.CreateAdminRequest{
			Login:    "newadmin",
			Name:     "New Admin",
			Password: "securepassword",
			Role:     models.RoleAdmin,
		}

		body, err := json.Marshal(reqDto)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		storeMock.EXPECT().CreateUser(mock.Anything, mock.AnythingOfType("*models.User")).RunAndReturn(
			func(ctx context.Context, u *models.User) error {
				assert.NotEmpty(t, u.HashedPassword)
				assert.Equal(t, "newadmin", u.Login)
				assert.Equal(t, models.RoleAdmin, u.Role)
				u.ID = uuid.New()
				return nil
			},
		)

		server.createUser(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var res dto.UserResponse
		err = json.NewDecoder(rr.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, "newadmin", res.Login)
		assert.Equal(t, models.RoleAdmin, res.Role)
	})

	t.Run("validation_failure", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		reqDto := dto.CreateAdminRequest{
			Login:    "ne",    // too short
			Name:     "N",     // too short
			Password: "short", // too short
			Role:     models.RoleAdmin,
		}

		body, err := json.Marshal(reqDto)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.createUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("bad_request_body", func(t *testing.T) {
		server, _, _, _, _ := newTestServer(t)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBufferString("{invalid json}"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.createUser(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("store_error", func(t *testing.T) {
		server, storeMock, _, _, _ := newTestServer(t)

		reqDto := dto.CreateAdminRequest{
			Login:    "newadmin",
			Name:     "New Admin",
			Password: "securepassword",
			Role:     models.RoleAdmin,
		}

		body, err := json.Marshal(reqDto)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		storeMock.EXPECT().CreateUser(mock.Anything, mock.AnythingOfType("*models.User")).Return(assert.AnError)

		server.createUser(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})
}
