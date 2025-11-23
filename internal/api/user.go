package api

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		access, err := getAccessCookie(r)
		if err != nil || access == "" {
			s.WriteError(w, r, errlocal.NewErrUnauthorized("missing or invalid authorization", err.Error(), nil))
			return
		}
		claims, err := s.authManager.Parse(access)
		if err != nil {
			s.WriteError(w, r, errlocal.NewErrUnauthorized("invalid token", err.Error(), nil))
			return
		}

		user := &models.User{
			ID:    uuid.MustParse(claims.UserID),
			Login: claims.Login,
		}
		ctx := utils.SetUser(r.Context(), user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) userMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUser := utils.GetUser(r.Context())
		if ctxUser == nil {
			s.WriteError(w, r, errlocal.NewErrUnauthorized("unauthorized", "user not found", nil))
			return
		}

		user, err := s.store.GetUser(r.Context(), ctxUser.ID, true)
		if err != nil {
			var notFoundErr *errlocal.ErrNotFound
			if errors.As(err, &notFoundErr) {
				s.WriteError(w, r, notFoundErr)
				return
			}

			s.WriteError(w, r, errlocal.NewErrInternal("failed to get user", err.Error(),
				map[string]any{"user_id": ctxUser.ID.String()}))
			return
		}

		ctx := utils.SetUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUser godoc
// @Summary Get user information
// @Description Get user details by ID (requires authentication)
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserResponse "User information"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/me [get]
func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	s.WriteResponse(w, r, http.StatusOK, utils.GetUser(r.Context()))
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user
// @Tags users
// @Accept json
// @Produce json
// @Param updateUserRequest body dto.UpdateUserRequest true "New user details"
// @Success 200 {object} dto.UserResponse "User information"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/me [patch]
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	body, err := dto.GetRequestBody[dto.UpdateUserRequest](r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid request body", err.Error(), nil))
		return
	}

	u := utils.GetUser(r.Context())
	u.Name = body.Name
	if err := s.store.UpdateUser(r.Context(), u); err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusOK, u)
}

// DeleteUser godoc
// @Summary Delete user account
// @Description Delete the authenticated user's account
// @Tags users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 204 "User deleted successfully"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Router /users/me [delete]
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	user := utils.GetUser(r.Context())

	if err := s.store.DeleteUser(r.Context(), user.ID); err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusNoContent, nil)
}

// changePassword godoc
// @Summary Change user password
// @Description Change the password of the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Param changePasswordRequest body dto.ChangePasswordRequest true "New password details"
// @Success 202 "Password changed successfully"
// @Failure 400 {object} errlocal.ErrBadRequest "Invalid request body"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/me/change-password [put]
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	b, err := dto.GetRequestBody[dto.ChangePasswordRequest](r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid request body", err.Error(), nil))
		return
	}

	user := utils.GetUser(r.Context())
	if err := utils.CompareHashPass(user.HashedPassword, b.OldPassword); err != nil {
		s.WriteError(w, r, errlocal.NewErrForbidden("old password does not match", "", nil))
		return
	}

	newHashedPass, _ := utils.HashPass(b.NewPassword)
	if err := s.store.UpdateUserPass(r.Context(), user.ID, newHashedPass); err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusAccepted, nil)
}

// Logout godoc
// @Summary Logout user
// @Description Revoke all tokens for the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Success 204 "User logged out successfully"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/me/logout [post]
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	user := utils.GetUser(r.Context())
	if err := s.authManager.RevokeAllUserTokens(r.Context(), user.ID); err != nil {
		s.WriteError(w, r, errlocal.NewErrInternal("failed to revoke tokens", err.Error(), nil))
		return
	}

	clearAuthCookies(w)
	s.WriteResponse(w, r, http.StatusNoContent, nil)
}

// SetAvatar godoc
// @Summary Set user avatar
// @Description Upload and set a new avatar for the authenticated user
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Avatar image file (JPEG or PNG, max 10MB)"
// @Success 202 {object} dto.UploadAvatarResponse "Avatar updated successfully"
// @Failure 400 {object} errlocal.ErrBadRequest "Invalid avatar file"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/me/avatar [put]
func (s *Server) setAvatar(w http.ResponseWriter, r *http.Request) {
	avatar, err := dto.GetAvatarFromMultipartForm(r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid avatar file", err.Error(), nil))
		return
	}
	defer func() {
		_ = avatar.Entry.Close()
	}()

	user := utils.GetUser(r.Context())
	if err := s.fileStore.UpdateAvatar(r.Context(), user, avatar); err != nil {
		s.WriteError(w, r, err)
		return
	}

	if err := s.store.UpdateAvatar(r.Context(), user); err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusAccepted, dto.UploadAvatarResponse{
		AvatarURL: *user.Avatar,
	})
}

// DeleteAvatar godoc
// @Summary Delete user avatar
// @Description Remove the avatar of the authenticated user
// @Tags users
// @Accept json
// @Produce json
// @Success 204 "Avatar deleted successfully"
// @Failure 400 {object} errlocal.ErrBadRequest "No avatar to delete"
// @Failure 401 {object} errlocal.ErrUnauthorized "Unauthorized"
// @Failure 403 {object} errlocal.ErrForbidden "Forbidden - user ID mismatch"
// @Failure 404 {object} errlocal.ErrNotFound "User not found"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Security BearerAuth
// @Router /users/me/avatar [delete]
func (s *Server) deleteAvatar(w http.ResponseWriter, r *http.Request) {
	user := utils.GetUser(r.Context())
	if user.Avatar == nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("no avatar to delete", "user has no avatar", nil))
		return
	}

	if err := s.fileStore.DeleteAvatar(r.Context(), *user.Avatar); err != nil {
		s.WriteError(w, r, err)
		return
	}

	user.Avatar = nil
	if err := s.store.UpdateAvatar(r.Context(), user); err != nil {
		s.WriteError(w, r, err)
		return
	}

	s.WriteResponse(w, r, http.StatusNoContent, nil)
}
