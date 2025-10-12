package api

import (
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func (s *Server) loginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := dto.GetRequestBody[dto.AuthRequest](r)
		if err != nil {
			s.WriteError(w, r, errlocal.NewErrBadRequest("invalid request body", err.Error(), nil))
			return
		}
		ctx := utils.SetRequestBody(r.Context(), b)

		existedUser, err := s.store.GetUserByLogin(ctx, b.Login)
		if err != nil {
			var notFoundErr *errlocal.ErrNotFound
			if errors.As(err, &notFoundErr) {
				ctx = utils.SetUser(ctx, b.ToModel())
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			s.WriteError(w, r, errlocal.NewErrInternal("failed check user", err.Error(),
				map[string]any{"login": b.Login}))
			return
		}

		ctx = utils.SetUser(ctx, *existedUser)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthRequest true "Login credentials"
// @Success 200 {object} dto.AuthResponse "Tokens for existing user"
// @Success 201 {object} dto.AuthResponse "Tokens for newly created user"
// @Failure 400 {object} errlocal.ErrBadRequest "Invalid request body"
// @Failure 401 {object} errlocal.ErrUnauthorized "Invalid credentials"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Router /login [post]
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var loginErr errlocal.LocalError
	var statusCode int
	defer func() {
		s.writeLoginHistory(r, statusCode, loginErr)
	}()

	u := utils.GetUser(r.Context()).(models.User)
	b := utils.GetRequestBody(r.Context()).(*dto.AuthRequest)

	if u.ID == uuid.Nil {
		if err := s.store.CreateUser(r.Context(), &u); err != nil {
			statusCode = http.StatusInternalServerError
			loginErr = errlocal.NewErrInternal("failed to create user", err.Error(),
				map[string]any{"login": u.Login})
			s.WriteError(w, r, loginErr)
			return
		}

		statusCode = http.StatusCreated
	} else {
		if err := utils.CompareHashPass(u.HashedPassword, b.Password); err != nil {
			statusCode = http.StatusUnauthorized
			loginErr = errlocal.NewErrUnauthorized("invalid credentials", "", nil)
			s.WriteError(w, r, loginErr)
			return
		}
		statusCode = http.StatusOK
	}

	tokens, tErr := s.authManager.CreateNewPair(r.Context(), u)
	if tErr != nil {
		statusCode = http.StatusInternalServerError
		loginErr = errlocal.NewErrInternal("failed to create tokens", tErr.Error(),
			map[string]any{"user_id": u.ID.String(), "login": u.Login})
		s.WriteError(w, r, loginErr)
		return
	}

	s.WriteResponse(w, r, statusCode, dto.NewAuthResponse(u, tokens.Access, tokens.Refresh))
}

// Refresh godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshRequest true "Refresh token"
// @Success 201 {object} dto.RefreshResponse "New tokens"
// @Failure 400 {object} errlocal.ErrBadRequest "Invalid request body"
// @Failure 401 {object} errlocal.ErrUnauthorized "Invalid or expired token"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Router /refresh [post]
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	b, err := dto.GetRequestBody[dto.RefreshRequest](r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("invalid token", err.Error(), nil))
		return
	}

	tokens, tErr := s.authManager.Refresh(r.Context(), b.RefreshToken)
	if tErr != nil {
		var notFoundErr *errlocal.ErrNotFound
		if errors.As(tErr, &notFoundErr) {
			s.WriteError(w, r, errlocal.NewErrUnauthorized("token not found", "", nil))
			return
		}
		if errors.Is(tErr, jwt.ErrTokenExpired) {
			s.WriteError(w, r, errlocal.NewErrUnauthorized("token expired", "", nil))
			return
		}
		s.WriteError(w, r, errlocal.NewErrInternal("failed to refresh tokens", tErr.Error(),
			map[string]any{"refresh_token": b.RefreshToken}))
		return
	}

	s.WriteResponse(w, r, http.StatusCreated, dto.NewRefreshResponse(tokens.Access, tokens.Refresh))
}
