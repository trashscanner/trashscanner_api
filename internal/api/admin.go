package api

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

// getUsersList godoc
// @Summary      Get users list
// @Description  Get paginated list of all users with their stats
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        offset  query     int  false  "Offset"            default(0)
// @Param        limit   query     int  false  "Limit"             default(100)
// @Success      200     {object}  dto.AdminUserListResponse
// @Failure      400     {object}  errlocal.ErrBadRequest
// @Failure      401     {object}  errlocal.ErrUnauthorized
// @Failure      403     {object}  errlocal.ErrForbidden
// @Failure      500     {object}  errlocal.ErrInternal
// @Router       /api/v1/admin/users [get]
func (s *Server) getUsersList(w http.ResponseWriter, r *http.Request) {
	limit := utils.GetQueryParam[int](r, limitQueryKey, defaultLimit)
	offset := utils.GetQueryParam[int](r, offsetQueryKey, defaultOffset)

	if limit == 0 {
		limit = defaultLimit
	}

	users, err := s.store.GetAdminUsers(r.Context(), int32(limit), int32(offset))
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	totalCount, err := s.store.CountUsers(r.Context())
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusOK, dto.NewAdminUserListResponse(users, totalCount, limit, offset))
}

// createUser godoc
// @Summary      Create user as admin
// @Description  Create a new user with specific role (e.g., admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        request body      dto.CreateAdminRequest true "User data"
// @Success      201     {object}  dto.UserResponse
// @Failure      400     {object}  errlocal.ErrBadRequest
// @Failure      401     {object}  errlocal.ErrUnauthorized
// @Failure      403     {object}  errlocal.ErrForbidden
// @Failure      409     {object}  errlocal.ErrConflict
// @Failure      500     {object}  errlocal.ErrInternal
// @Router       /api/v1/admin/users [post]
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	req, err := dto.GetRequestBody[dto.CreateAdminRequest](r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid request body or validation failed", err.Error(), nil))
		return
	}

	model := req.ToModel()

	if err := s.store.CreateUser(r.Context(), model); err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusCreated, dto.UserResponse(*model))
}

// getAdminUser godoc
// @Summary      Get user by ID
// @Description  Get a single user with their stats and predictions (scans)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Param        user_id path      string true  "User ID (UUID)"
// @Param        offset  query     int    false "Predictions offset" default(0)
// @Param        limit   query     int    false "Predictions limit"  default(100)
// @Success      200     {object}  dto.AdminUserDetailResponse
// @Failure      400     {object}  errlocal.ErrBadRequest
// @Failure      401     {object}  errlocal.ErrUnauthorized
// @Failure      403     {object}  errlocal.ErrForbidden
// @Failure      404     {object}  errlocal.ErrNotFound
// @Failure      500     {object}  errlocal.ErrInternal
// @Router       /api/v1/admin/users/{user_id} [get]
func (s *Server) getAdminUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(mux.Vars(r)[userIDTag])
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid user ID", err.Error(), nil))
		return
	}

	user, err := s.store.GetAdminUserByID(r.Context(), userID)
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	limit := utils.GetQueryParam[int](r, limitQueryKey, defaultLimit)
	offset := utils.GetQueryParam[int](r, offsetQueryKey, defaultOffset)

	if limit == 0 {
		limit = defaultLimit
	}

	predictions, err := s.store.GetPredictionsByUserID(r.Context(), userID, offset, limit)
	if err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusOK, dto.NewAdminUserDetailResponse(*user, predictions, limit, offset))
}
