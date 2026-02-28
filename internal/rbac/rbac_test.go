package rbac

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name          string
		userRole      models.Role
		allowedRoles  []models.Role
		expectedCode  int
		expectedError bool
	}{
		{
			name:          "admin allowed",
			userRole:      models.RoleAdmin,
			allowedRoles:  []models.Role{models.RoleAdmin},
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
		{
			name:          "user denied",
			userRole:      models.RoleUser,
			allowedRoles:  []models.Role{models.RoleAdmin},
			expectedCode:  http.StatusForbidden,
			expectedError: true,
		},
		{
			name:          "user allowed alongside admin",
			userRole:      models.RoleUser,
			allowedRoles:  []models.Role{models.RoleUser, models.RoleAdmin},
			expectedCode:  http.StatusOK,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeErrCalled := false
			writeErr := func(w http.ResponseWriter, r *http.Request, err error) {
				writeErrCalled = true
				var localErr errlocal.LocalError
				if errors.As(err, &localErr) {
					w.WriteHeader(localErr.Code())
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
			}

			user := &models.User{
				ID:   uuid.New(),
				Role: tt.userRole,
			}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req = req.WithContext(utils.SetUser(req.Context(), user))

			mw := RequireRole(writeErr, tt.allowedRoles...)
			nextCalled := false
			h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)
			assert.Equal(t, tt.expectedError, writeErrCalled)
			assert.Equal(t, !tt.expectedError, nextCalled)
		})
	}
}
