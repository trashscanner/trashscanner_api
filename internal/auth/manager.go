package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/store"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type TokenPair struct {
	Access, Refresh string
}

type AuthManager interface {
	CreateNewPair(ctx context.Context, user models.User) (*TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, error)
	Parse(tokenStr string) (*Claims, error)
}

type jwtManager struct {
	generator *jwtGenerator
	store     store.Store
}

func NewJWTManager(cfg config.Config, store store.Store) (AuthManager, error) {
	generator, err := newJWTGenerator(cfg)
	if err != nil {
		return nil, err
	}

	return &jwtManager{
		generator: generator,
		store:     store,
	}, nil
}

func (m *jwtManager) CreateNewPair(ctx context.Context, user models.User) (*TokenPair, error) {
	tokens, err := m.generator.newPair(user)
	if err != nil {
		return nil, err
	}

	refresh, err := m.generator.parseRefresh(tokens.Refresh)
	if err != nil {
		return nil, err
	}

	if err := m.store.RevokeAllUserTokens(ctx, user.ID); err != nil {
		return nil, err
	}

	if err := m.store.InsertRefreshToken(ctx, refresh); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (m *jwtManager) Refresh(ctx context.Context, refreshTokenStr string) (*TokenPair, error) {
	parsedToken, err := m.generator.parseRefresh(refreshTokenStr)
	if err != nil {
		return nil, err
	}

	user, err := m.store.GetUser(ctx, parsedToken.UserID, false)
	if err != nil {
		return nil, err
	}

	storedToken, err := m.store.GetRefreshTokenByHash(ctx, utils.HashToken(refreshTokenStr))
	if err != nil {
		return nil, err
	}
	if storedToken.Revoked {
		return nil, fmt.Errorf("token revoked")
	}
	if storedToken.ExpiresAt.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	return m.CreateNewPair(ctx, *user)
}

func (m *jwtManager) Parse(tokenStr string) (*Claims, error) {
	return m.generator.parseAccess(tokenStr)
}
