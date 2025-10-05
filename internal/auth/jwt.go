package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type jwtGenerator struct {
	signingMethod         jwt.SigningMethod
	privateKey, publicKey interface{}
	ttlAccess, ttlRefresh time.Duration
}

func newJWTGenerator(cfg config.Config) (*jwtGenerator, error) {
	privateKey, publicKey, err := utils.GetEdDSAKeysFromEnv()
	if err != nil {
		return nil, err
	}
	return &jwtGenerator{
		signingMethod: jwt.GetSigningMethod(cfg.Auth.Algorithm),
		privateKey:    privateKey,
		publicKey:     publicKey,
		ttlAccess:     cfg.Auth.AccessTokenTTL,
		ttlRefresh:    cfg.Auth.RefreshTokenTTL,
	}, nil
}

type Claims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id" validate:"required,uuid"`
	Login     string `json:"login"`
	TokenType string `json:"token_type"`
}

func (m *jwtGenerator) newPair(user models.User) (*TokenPair, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(m.signingMethod, Claims{
		UserID:    user.ID.String(),
		Login:     user.Login,
		TokenType: "access",

		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttlAccess)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	})
	accessString, err := accessToken.SignedString(m.privateKey)
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.NewWithClaims(m.signingMethod, jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttlRefresh)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ID:        uuid.New().String(),
	})

	refreshString, err := refreshToken.SignedString(m.privateKey)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		Access:  accessString,
		Refresh: refreshString,
	}, nil
}

func (m *jwtGenerator) parseAccess(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != m.signingMethod.Alg() {
			return nil, jwt.ErrTokenUnverifiable
		}

		return m.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	if claims.TokenType != "access" {
		return nil, jwt.ErrTokenUnverifiable
	}

	return claims, nil
}

func (m *jwtGenerator) parseRefresh(tokenStr string) (*models.RefreshToken, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != m.signingMethod.Alg() {
			return nil, jwt.ErrTokenUnverifiable
		}

		return m.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	if claims.Subject == "" {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return models.NewRefreshFromClaims(utils.HashToken(tokenStr), *claims), nil
}
