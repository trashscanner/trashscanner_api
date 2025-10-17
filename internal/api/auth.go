package api

import (
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/api/dto"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

func (s *Server) loginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := dto.GetRequestBody[dto.LoginUserRequest](r)
		if err != nil {
			s.WriteError(w, r, errlocal.NewErrBadRequest("invalid request body", err.Error(), nil))
			return
		}
		ctx := utils.SetRequestBody(r.Context(), b)
		requestUser := b.ToModel()

		existedUser, err := s.store.GetUserByLogin(ctx, requestUser.Login)
		if err != nil {
			var notFoundErr *errlocal.ErrNotFound
			if errors.As(err, &notFoundErr) {
				ctx = utils.SetUser(ctx, &requestUser)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			s.WriteError(w, r, errlocal.NewErrInternal("failed check user", err.Error(),
				map[string]any{"login": requestUser.Login}))
			return
		}

		ctx = utils.SetUser(ctx, existedUser)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Login godoc
// @Summary User registration
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginUserRequest true "Login credentials"
// @Success 201 {object} dto.AuthResponse "User registered and tokens returned"
// @Failure 400 {object} errlocal.ErrBadRequest "Invalid request body"
// @Failure 401 {object} errlocal.ErrUnauthorized "Invalid credentials"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Router /register [post]
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := utils.GetUser(ctx)

	if u.ID != uuid.Nil {
		s.WriteError(w, r, errlocal.NewErrConflict("user with this login already exists", "", nil))
		return
	}

	if u.Name == "" {
		s.WriteError(w, r, errlocal.NewErrBadRequest("name is required", "", nil))
		return
	}

	if err := s.store.CreateUser(ctx, u); err != nil {
		s.WriteError(w, r, err)
		return
	}

	tokens, tokenErr := s.authManager.CreateNewPair(ctx, *u)
	if tokenErr != nil {
		s.WriteError(w, r, errlocal.NewErrInternal("error create tokens", tokenErr.Error(), nil))
		s.writeLoginHistory(r, http.StatusInternalServerError, tokenErr)
		return
	}

	setAuthCookies(w, tokens)
	s.WriteResponse(w, r, http.StatusCreated, dto.NewAuthResponse(*u, tokens.Access, tokens.Refresh))
	s.writeLoginHistory(r, http.StatusCreated, nil)
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthRequest true "Login credentials"
// @Success 200 {object} dto.AuthResponse "Tokens for existing user"
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

	u := utils.GetUser(r.Context())
	if u.ID == uuid.Nil {
		s.WriteError(w, r, errlocal.NewErrUnauthorized("user not found, please create an account", "", nil))
		return
	}

	b := utils.GetRequestBody(r.Context()).(*dto.LoginUserRequest)
	if err := utils.CompareHashPass(u.HashedPassword, b.Password); err != nil {
		statusCode = http.StatusUnauthorized
		loginErr = errlocal.NewErrUnauthorized("invalid credentials", "", nil)
		s.WriteError(w, r, loginErr)
		return
	}

	statusCode = http.StatusOK

	tokens, tErr := s.authManager.CreateNewPair(r.Context(), *u)
	if tErr != nil {
		statusCode = http.StatusInternalServerError
		loginErr = errlocal.NewErrInternal("failed to create tokens", tErr.Error(),
			map[string]any{"user_id": u.ID.String(), "login": u.Login})
		s.WriteError(w, r, loginErr)
		return
	}

	setAuthCookies(w, tokens)
	s.WriteResponse(w, r, statusCode, dto.NewAuthResponse(*u, tokens.Access, tokens.Refresh))
}

// Refresh godoc
// @Summary Refresh access token
// @Description Get new access token using refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Success 201 "New access token set in HttpOnly cookie"
// @Failure 400 {object} errlocal.ErrBadRequest "Invalid request body"
// @Failure 401 {object} errlocal.ErrUnauthorized "Invalid or expired token"
// @Failure 500 {object} errlocal.ErrInternal "Internal server error"
// @Router /refresh [post]
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := getRefreshFromCookie(r)
	if err != nil {
		s.WriteError(w, r, errlocal.NewErrBadRequest("missing refresh token cookie", err.Error(), nil))
		return
	}

	tokens, tErr := s.authManager.Refresh(r.Context(), refreshToken)
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
			nil))
		return
	}

	setAuthCookies(w, tokens)
	s.WriteResponse(w, r, http.StatusAccepted, nil)
}
