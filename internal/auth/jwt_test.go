package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/utils"
)

type JWTManagerSuite struct {
	suite.Suite
	manager *JWTManager
	user    models.User
}

func (s *JWTManagerSuite) SetupSuite() {
	utils.GenerateAndSetKeys()

	cfg := config.Config{
		Auth: config.AuthManagerConfig{
			Algorithm:       "EdDSA",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}

	var err error
	s.manager, err = NewJWTManager(cfg)
	s.Require().NoError(err)

	s.user = models.User{
		ID:    uuid.New(),
		Login: "testuser",
	}
}

func TestJWTManagerSuite(t *testing.T) {
	suite.Run(t, new(JWTManagerSuite))
}

func (s *JWTManagerSuite) TestNewPair_Success() {
	tokens, err := s.manager.NewPair(s.user)

	s.NoError(err)
	s.NotEmpty(tokens.Access)
	s.NotEmpty(tokens.Refresh)
	s.NotEmpty(tokens.TokenFamily)

	_, err = uuid.Parse(tokens.TokenFamily)
	s.NoError(err)
}

func (s *JWTManagerSuite) TestNewPair_DifferentTokens() {
	tokens, err := s.manager.NewPair(s.user)

	s.NoError(err)
	s.NotEqual(tokens.Access, tokens.Refresh)
}

func (s *JWTManagerSuite) TestNewPair_DifferentTokenFamilies() {
	tokens1, err := s.manager.NewPair(s.user)
	s.NoError(err)

	tokens2, err := s.manager.NewPair(s.user)
	s.NoError(err)

	s.NotEqual(tokens1.TokenFamily, tokens2.TokenFamily)
}

func (s *JWTManagerSuite) TestParseAccess_ValidToken() {
	tokens, err := s.manager.NewPair(s.user)
	s.NoError(err)

	claims, err := s.manager.ParseAccess(tokens.Access)

	s.NoError(err)
	s.Equal(s.user.ID.String(), claims.UserID)
	s.Equal(s.user.Login, claims.Login)
	s.Equal("access", claims.TokenType)
}

func (s *JWTManagerSuite) TestParseAccess_RejectRefreshToken() {
	tokens, err := s.manager.NewPair(s.user)
	s.NoError(err)

	_, err = s.manager.ParseAccess(tokens.Refresh)

	s.Error(err)
}

func (s *JWTManagerSuite) TestParseAccess_InvalidToken() {
	_, err := s.manager.ParseAccess("invalid.token.here")

	s.Error(err)
}

func (s *JWTManagerSuite) TestParseAccess_WrongSignature() {
	utils.GenerateAndSetKeys()
	wrongCfg := config.Config{
		Auth: config.AuthManagerConfig{
			Algorithm:       "EdDSA",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
	}
	wrongManager, err := NewJWTManager(wrongCfg)
	s.NoError(err)

	tokens, err := wrongManager.NewPair(s.user)
	s.NoError(err)

	_, err = s.manager.ParseAccess(tokens.Access)

	s.Error(err)
}

func (s *JWTManagerSuite) TestParseRefresh_ValidToken() {
	tokens, err := s.manager.NewPair(s.user)
	s.NoError(err)

	token, err := s.manager.ParseRefresh(tokens.Refresh)

	s.NoError(err)
	s.NotNil(token)
	s.True(token.Valid)
}

func (s *JWTManagerSuite) TestParseRefresh_AccessTokenAllowed() {
	tokens, err := s.manager.NewPair(s.user)
	s.NoError(err)

	token, err := s.manager.ParseRefresh(tokens.Access)

	s.NoError(err)
	s.NotNil(token)
}

func (s *JWTManagerSuite) TestParseRefresh_InvalidToken() {
	_, err := s.manager.ParseRefresh("invalid.token.here")

	s.Error(err)
}

func (s *JWTManagerSuite) TestGetTokenFamily_Success() {
	tokens, err := s.manager.NewPair(s.user)
	s.NoError(err)

	family, err := s.manager.GetTokenFamily(tokens.Refresh)

	s.NoError(err)
	s.Equal(tokens.TokenFamily, family)

	_, err = uuid.Parse(family)
	s.NoError(err)
}

func (s *JWTManagerSuite) TestGetTokenFamily_InvalidToken() {
	_, err := s.manager.GetTokenFamily("invalid.token.here")

	s.Error(err)
}

func (s *JWTManagerSuite) TestTokenExpiration_AccessToken() {
	utils.GenerateAndSetKeys()
	shortCfg := config.Config{
		Auth: config.AuthManagerConfig{
			Algorithm:       "EdDSA",
			AccessTokenTTL:  1 * time.Second,
			RefreshTokenTTL: 2 * time.Second,
		},
	}
	shortManager, err := NewJWTManager(shortCfg)
	s.NoError(err)

	tokens, err := shortManager.NewPair(s.user)
	s.NoError(err)

	_, err = shortManager.ParseAccess(tokens.Access)
	s.NoError(err)

	time.Sleep(2 * time.Second)

	_, err = shortManager.ParseAccess(tokens.Access)
	s.Error(err)
}

func (s *JWTManagerSuite) TestTokenExpiration_RefreshToken() {
	utils.GenerateAndSetKeys()
	shortCfg := config.Config{
		Auth: config.AuthManagerConfig{
			Algorithm:       "EdDSA",
			AccessTokenTTL:  1 * time.Second,
			RefreshTokenTTL: 2 * time.Second,
		},
	}
	shortManager, err := NewJWTManager(shortCfg)
	s.NoError(err)

	tokens, err := shortManager.NewPair(s.user)
	s.NoError(err)

	_, err = shortManager.ParseRefresh(tokens.Refresh)
	s.NoError(err)

	time.Sleep(3 * time.Second)

	_, err = shortManager.ParseRefresh(tokens.Refresh)
	s.Error(err)
}
