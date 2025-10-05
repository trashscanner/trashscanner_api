package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

const (
	authHeaderPrefix = "Bearer "
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), authHeaderPrefix)
		if token == "" {
			s.WriteError(w, errlocal.NewErrUnauthorized("missing or invalid authorization", "", nil))
			return
		}

		claims, err := s.authManager.Parse(token)
		if err != nil {
			s.WriteError(w, errlocal.NewErrUnauthorized("invalid token", err.Error(), nil))
			return
		}

		user := models.User{
			ID:    uuid.MustParse(claims.UserID),
			Login: claims.Login,
		}
		ctx := utils.SetUser(r.Context(), user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) userMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)[userKey]
		ctxUser, ok := utils.GetUser(r.Context()).(models.User)
		if ok {
			if ctxUser.ID.String() != userID {
				s.WriteError(w, errlocal.NewErrForbidden("forbidden", "user ID does not match", nil))
				return
			}
		}

		user, err := s.store.GetUser(r.Context(), uuid.MustParse(userID), true)
		if err != nil {
			var notFoundErr *errlocal.ErrNotFound
			if errors.As(err, &notFoundErr) {
				s.WriteError(w, notFoundErr)
				return
			}

			s.WriteError(w, errlocal.NewErrInternal("failed to get user", err.Error(),
				map[string]any{"user_id": userID}))
			return
		}

		ctx := utils.SetUser(r.Context(), *user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser godoc
// @Summary Get user information
// @Description Get user details by ID (requires authentication)
// @Tags users
// @Accept json
// @Produce json
// @Param user-id path string true "User ID"
// @Success 200 {object} dto.UserResponse "User information"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/{user-id} [get]
func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	user := utils.GetUser(r.Context()).(models.User)
	s.WriteResponse(w, http.StatusOK, user)
}
