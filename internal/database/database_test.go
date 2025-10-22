package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	su "github.com/stretchr/testify/suite"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
)

var testStore db.Querier

type databaseTestSuite struct {
	su.Suite
	ctx   context.Context
	store db.Querier
}

func TestMain(m *testing.M) {
	os.Setenv("CONFIG_PATH", "../../config/dev")
	os.Setenv("DATABASE_MIGRATIONS_PATH", "migrations")
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dsn := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", cfg.DB.Host, cfg.DB.Port),
		User:   url.UserPassword(cfg.DB.User, cfg.DB.Password),
		Path:   "/" + cfg.DB.Name,
	}

	ctx := context.Background()
	conn, err := pgxpool.New(ctx, dsn.String())
	if err != nil {
		log.Fatalf("failed to create connection pool: %v", err)
	}
	defer conn.Close()

	if err := RunMigrations(conn, cfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	testStore = db.New(conn)

	code := m.Run()

	if err := DownMigrations(conn, cfg); err != nil {
		log.Fatalf("failed to down migrations: %v", err)
	}

	os.Exit(code)
}

func TestDatabaseSuite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suite := &databaseTestSuite{
		ctx:   ctx,
		store: testStore,
	}

	su.Run(t, suite)
}

func (s *databaseTestSuite) createTestUser(login string) uuid.UUID {
	userID, err := s.store.CreateUser(s.ctx, db.CreateUserParams{
		Login:          login,
		HashedPassword: "hashedpassword",
	})
	s.Require().NoError(err)
	s.Require().NotEqual(uuid.Nil, userID)
	return userID
}

func (s *databaseTestSuite) TestCreateUser() {
	userID := s.createTestUser("testCreateUser")

	user, err := s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.Equal(userID, user.ID)
	s.Equal("testCreateUser", user.Login)
	s.Equal("hashedpassword", user.HashedPassword)
	s.False(user.Deleted)
}

func (s *databaseTestSuite) TestGetUserByID() {
	userID := s.createTestUser("testGetUserByID")

	user, err := s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.Equal(userID, user.ID)
	s.Equal("testGetUserByID", user.Login)
}

func (s *databaseTestSuite) TestGetUserByID_NotFound() {
	_, err := s.store.GetUserByID(s.ctx, uuid.New())
	s.Error(err)
}

func (s *databaseTestSuite) TestGetUserByLogin() {
	userID := s.createTestUser("testGetUserByLogin")

	user, err := s.store.GetUserByLogin(s.ctx, "testGetUserByLogin")
	s.NoError(err)
	s.Equal(userID, user.ID)
	s.Equal("testGetUserByLogin", user.Login)
}

func (s *databaseTestSuite) TestGetUserByLogin_NotFound() {
	_, err := s.store.GetUserByLogin(s.ctx, "nonexistent_user")
	s.Error(err)
}

func (s *databaseTestSuite) TestUpdateUserPassword() {
	userID := s.createTestUser("testUpdatePassword")

	err := s.store.UpdateUserPassword(s.ctx, db.UpdateUserPasswordParams{
		ID:             userID,
		HashedPassword: "new_hashed_password",
	})
	s.NoError(err)

	user, err := s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.Equal("new_hashed_password", user.HashedPassword)
}

func (s *databaseTestSuite) TestUpdateUserAvatar() {
	userID := s.createTestUser("testUpdateAvatar")

	user, err := s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.Nil(user.Avatar, "Avatar should be empty initially")

	avatarURL := "https://example.com/avatar.jpg"
	err = s.store.UpdateUserAvatar(s.ctx, db.UpdateUserAvatarParams{
		ID:     userID,
		Avatar: &avatarURL,
	})
	s.NoError(err)

	user, err = s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.NotNil(user.Avatar)
	s.Equal(avatarURL, *user.Avatar)

	newAvatarURL := "https://cdn.example.com/avatars/user123.png"
	err = s.store.UpdateUserAvatar(s.ctx, db.UpdateUserAvatarParams{
		ID:     userID,
		Avatar: &newAvatarURL,
	})
	s.NoError(err)

	user, err = s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.NotNil(user.Avatar)
	s.Equal(newAvatarURL, *user.Avatar)
}

func (s *databaseTestSuite) TestUpdateUserAvatar_Clear() {
	userID := s.createTestUser("testClearAvatar")

	initialURL := "https://example.com/initial.jpg"
	err := s.store.UpdateUserAvatar(s.ctx, db.UpdateUserAvatarParams{
		ID:     userID,
		Avatar: &initialURL,
	})
	s.NoError(err)

	user, err := s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.NotNil(user.Avatar)

	err = s.store.UpdateUserAvatar(s.ctx, db.UpdateUserAvatarParams{
		ID:     userID,
		Avatar: nil,
	})
	s.NoError(err)

	user, err = s.store.GetUserByID(s.ctx, userID)
	s.NoError(err)
	s.Nil(user.Avatar, "Avatar should be cleared")
}

func (s *databaseTestSuite) TestDeleteUser() {
	userID := s.createTestUser("testDeleteUser")

	err := s.store.DeleteUser(s.ctx, userID)
	s.NoError(err)

	_, err = s.store.GetUserByID(s.ctx, userID)
	s.Error(err)
}

func (s *databaseTestSuite) TestCreateRefreshToken() {
	userID := s.createTestUser("testCreateRefreshToken")

	tokenID, err := s.store.CreateRefreshToken(s.ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: "test_hash_123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.NoError(err)
	s.NotEqual(uuid.Nil, tokenID)
}

func (s *databaseTestSuite) TestGetRefreshTokenByHash() {
	userID := s.createTestUser("testGetRefreshTokenByHash")

	_, err := s.store.CreateRefreshToken(s.ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: "unique_hash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.NoError(err)

	token, err := s.store.GetRefreshTokenByHash(s.ctx, "unique_hash")
	s.NoError(err)
	s.Equal(userID, token.UserID)
	s.Equal("unique_hash", token.TokenHash)
	s.False(token.Revoked)
}

func (s *databaseTestSuite) TestGetRefreshTokenByHash_NotFound() {
	_, err := s.store.GetRefreshTokenByHash(s.ctx, "nonexistent_hash")
	s.Error(err)
}

func (s *databaseTestSuite) TestGetRefreshTokenByHash_Expired() {
	userID := s.createTestUser("testExpiredToken")

	_, err := s.store.CreateRefreshToken(s.ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: "expired_hash",
		ExpiresAt: time.Now().Add(-24 * time.Hour),
	})
	s.NoError(err)

	_, err = s.store.GetRefreshTokenByHash(s.ctx, "expired_hash")
	s.Error(err)
}

func (s *databaseTestSuite) TestGetActiveTokensByUser() {
	userID := s.createTestUser("testGetActiveTokens")

	for i := 0; i < 3; i++ {
		_, err := s.store.CreateRefreshToken(s.ctx, db.CreateRefreshTokenParams{
			UserID:    userID,
			TokenHash: fmt.Sprintf("active_hash_%d", i),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		s.NoError(err)
	}

	tokens, err := s.store.GetActiveTokensByUser(s.ctx, userID)
	s.NoError(err)
	s.Len(tokens, 3)
}

func (s *databaseTestSuite) TestGetActiveTokensByUser_NoTokens() {
	userID := s.createTestUser("testNoActiveTokens")

	tokens, err := s.store.GetActiveTokensByUser(s.ctx, userID)
	s.NoError(err)
	s.Empty(tokens)
}

func (s *databaseTestSuite) TestRevokeAllUserTokens() {
	userID := s.createTestUser("testRevokeAll")

	for i := 0; i < 3; i++ {
		_, err := s.store.CreateRefreshToken(s.ctx, db.CreateRefreshTokenParams{
			UserID:    userID,
			TokenHash: fmt.Sprintf("revoke_hash_%d", i),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		s.NoError(err)
	}

	err := s.store.RevokeAllUserTokens(s.ctx, userID)
	s.NoError(err)

	tokens, err := s.store.GetActiveTokensByUser(s.ctx, userID)
	s.NoError(err)
	s.Empty(tokens)
}

func (s *databaseTestSuite) TestCreateLoginHistory() {
	userID := s.createTestUser("testCreateLoginHistory")

	historyID, err := s.store.CreateLoginHistory(s.ctx, db.CreateLoginHistoryParams{
		UserID:  userID,
		Success: true,
	})
	s.NoError(err)
	s.NotEqual(uuid.Nil, historyID)

	userAgent := "Mozilla/5.0"
	location := "Moscow, Russia"

	historyID2, err := s.store.CreateLoginHistory(s.ctx, db.CreateLoginHistoryParams{
		UserID:  userID,
		Success: false,
		FailureReason: func() *string {
			s := "Invalid password"
			return &s
		}(),
		UserAgent: &userAgent,
		Location:  &location,
	})
	s.NoError(err)
	s.NotEqual(uuid.Nil, historyID2)
}

func (s *databaseTestSuite) TestGetLoginHistoryByUser() {
	userID := s.createTestUser("testGetLoginHistory")

	for i := 0; i < 5; i++ {
		_, err := s.store.CreateLoginHistory(s.ctx, db.CreateLoginHistoryParams{
			UserID:  userID,
			Success: i%2 == 0,
		})
		s.NoError(err)
		time.Sleep(10 * time.Millisecond)
	}

	history, err := s.store.GetLoginHistoryByUser(s.ctx, db.GetLoginHistoryByUserParams{
		UserID: userID,
		Limit:  3,
		Offset: 0,
	})
	s.NoError(err)
	s.Len(history, 3)

	if len(history) > 1 {
		s.True(history[0].CreatedAt.After(history[1].CreatedAt))
	}
}

func (s *databaseTestSuite) TestGetLoginHistoryByUser_NoHistory() {
	userID := s.createTestUser("testNoLoginHistory")

	history, err := s.store.GetLoginHistoryByUser(s.ctx, db.GetLoginHistoryByUserParams{
		UserID: userID,
		Limit:  10,
		Offset: 0,
	})
	s.NoError(err)
	s.Empty(history)
}

func (s *databaseTestSuite) TestCreateStats() {
	userID := s.createTestUser("testCreateStats")

	statsID, err := s.store.CreateStats(s.ctx, userID)
	s.NoError(err)
	s.NotEqual(uuid.Nil, statsID)

	stats, err := s.store.GetStatsByUserID(s.ctx, userID)
	s.NoError(err)
	s.Equal(userID, stats.UserID)
	s.Equal("newbie", stats.Status)
	s.Equal(int32(0), stats.Rating)
	s.Equal(int32(0), stats.FilesScanned)
	s.Equal(0.0, stats.TotalWeight)
	s.False(stats.CreatedAt.IsZero())
	s.False(stats.UpdatedAt.IsZero())
}

func (s *databaseTestSuite) TestGetStatsByUserID() {
	userID := s.createTestUser("testGetStats")

	_, err := s.store.CreateStats(s.ctx, userID)
	s.NoError(err)

	stats, err := s.store.GetStatsByUserID(s.ctx, userID)
	s.NoError(err)
	s.Equal(userID, stats.UserID)
	s.Equal("newbie", stats.Status)
}

func (s *databaseTestSuite) TestGetStatsByUserID_NotFound() {
	_, err := s.store.GetStatsByUserID(s.ctx, uuid.New())
	s.Error(err)
}

func (s *databaseTestSuite) TestUpdateStats() {
	userID := s.createTestUser("testUpdateStats")

	_, err := s.store.CreateStats(s.ctx, userID)
	s.NoError(err)

	err = s.store.UpdateStats(s.ctx, db.UpdateStatsParams{
		UserID:       userID,
		Status:       "eco_warrior",
		Rating:       150,
		FilesScanned: 25,
		TotalWeight:  75.5,
	})
	s.NoError(err)

	stats, err := s.store.GetStatsByUserID(s.ctx, userID)
	s.NoError(err)
	s.Equal("eco_warrior", stats.Status)
	s.Equal(int32(150), stats.Rating)
	s.Equal(int32(25), stats.FilesScanned)
	s.Equal(75.5, stats.TotalWeight)
}

func (s *databaseTestSuite) TestDuplicateStatsForUser() {
	userID := s.createTestUser("testDuplicateStats")

	_, err := s.store.CreateStats(s.ctx, userID)
	s.NoError(err)

	_, err = s.store.CreateStats(s.ctx, userID)
	s.Error(err)
}

func (s *databaseTestSuite) createTestPrediction(userID uuid.UUID) uuid.UUID {
	testImgUrl := "/test/scan/" + uuid.NewString()
	status := "processing"

	id, err := s.store.CreateNewPrediction(s.ctx, db.CreateNewPredictionParams{
		UserID:    userID,
		TrashScan: testImgUrl,
		Status:    status,
	})
	s.NoError(err)
	s.NotZero(id)

	newPrediction, err := s.store.GetPrediction(s.ctx, id)
	s.NoError(err)
	s.Equal(id, newPrediction.ID)
	s.Equal(userID, newPrediction.UserID)
	s.NotZero(newPrediction.CreatedAt)
	s.NotZero(newPrediction.UpdatedAt)

	return id
}

func (s *databaseTestSuite) TestCompletePrediction() {
	userID := s.createTestUser("testCompletePrediction")
	predictionID := s.createTestPrediction(userID)

	result := "plastic"

	err := s.store.CompletePrediction(s.ctx, db.CompletePredictionParams{
		Status: "completed",
		Result: &result,
		ID:     predictionID,
	})
	s.NoError(err)

	updated, err := s.store.GetPrediction(s.ctx, predictionID)
	s.NoError(err)
	s.Equal("completed", updated.Status)
	s.Equal("plastic", *updated.Result)
	s.Zero(updated.Error)
}

func (s *databaseTestSuite) TestListPredictions() {
	userID := s.createTestUser("testCompletePrediction")
	predictions := make(uuid.UUIDs, 5)
	for i := range 5 {
		predictions[i] = s.createTestPrediction(userID)
	}

	existed, err := s.store.GetPredictionsByUserID(s.ctx, db.GetPredictionsByUserIDParams{
		UserID: userID,
		Limit:  10,
		Offset: 0,
	})
	s.NoError(err)
	s.Len(existed, 5)
	existedIDs := make(uuid.UUIDs, 5)
	for i, ex := range existed {
		existedIDs[i] = ex.ID
	}

	s.ElementsMatch(predictions, existedIDs)
}
