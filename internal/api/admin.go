package api

import (
	"net/http"

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

	if limit <= 0 {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = defaultOffset
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
