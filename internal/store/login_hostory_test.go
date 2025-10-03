package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestInsertLoginHistory(t *testing.T) {
	t.Run("Insert successful login history", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		loginHistory := models.LoginHistory{
			UserID:       testdata.User1ID,
			LoginAttempt: testdata.User1.Login,
			Success:      true,
			IpAddress:    testdata.LoginHistory1.IpAddress,
			UserAgent:    testdata.LoginHistory1.UserAgent,
			Location:     testdata.LoginHistory1.Location,
		}

		mockQ.EXPECT().CreateLoginHistory(mock.Anything, db.CreateLoginHistoryParams{
			UserID:        loginHistory.UserID,
			LoginAttempt:  loginHistory.LoginAttempt,
			Success:       loginHistory.Success,
			FailureReason: loginHistory.FailureReason,
			IpAddress:     loginHistory.IpAddress,
			UserAgent:     loginHistory.UserAgent,
			Location:      loginHistory.Location,
		}).Return(testdata.LoginHistory1ID, nil).Once()

		err := store.InsertLoginHistory(ctx, testdata.User1ID, &loginHistory)

		assert.NoError(t, err)
		assert.Equal(t, testdata.LoginHistory1ID, loginHistory.ID)
	})

	t.Run("Insert failed login history", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		failureReason := "Invalid password"
		loginHistory := models.LoginHistory{
			UserID:        testdata.User1ID,
			LoginAttempt:  testdata.User1.Login,
			Success:       false,
			FailureReason: &failureReason,
			IpAddress:     testdata.LoginHistory2.IpAddress,
			UserAgent:     testdata.LoginHistory2.UserAgent,
			Location:      testdata.LoginHistory2.Location,
		}

		mockQ.EXPECT().CreateLoginHistory(mock.Anything, mock.MatchedBy(func(params db.CreateLoginHistoryParams) bool {
			return params.UserID == testdata.User1ID &&
				params.LoginAttempt == testdata.User1.Login &&
				params.Success == false &&
				params.FailureReason != nil &&
				*params.FailureReason == failureReason
		})).Return(testdata.LoginHistory2ID, nil).Once()

		err := store.InsertLoginHistory(ctx, testdata.User1ID, &loginHistory)

		assert.NoError(t, err)
		assert.Equal(t, testdata.LoginHistory2ID, loginHistory.ID)
	})

	t.Run("Insert login history with minimal data", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		newID := uuid.New()
		loginHistory := models.LoginHistory{
			UserID:       testdata.User2ID,
			LoginAttempt: testdata.User2.Login,
			Success:      true,
		}

		mockQ.EXPECT().CreateLoginHistory(mock.Anything, db.CreateLoginHistoryParams{
			UserID:       testdata.User2ID,
			LoginAttempt: testdata.User2.Login,
			Success:      true,
		}).Return(newID, nil).Once()

		err := store.InsertLoginHistory(ctx, testdata.User2ID, &loginHistory)

		assert.NoError(t, err)
		assert.Equal(t, newID, loginHistory.ID)
	})

	t.Run("Insert login history fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		loginHistory := testdata.LoginHistory1
		insertErr := assert.AnError

		mockQ.EXPECT().CreateLoginHistory(mock.Anything, mock.Anything).
			Return(uuid.Nil, insertErr).Once()

		err := store.InsertLoginHistory(ctx, testdata.User1ID, &loginHistory)

		assert.ErrorIs(t, err, insertErr)
	})
}

func TestGetLoginHistory(t *testing.T) {
	t.Run("Get login history for user with single page", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbHistory := []db.LoginHistory{
			testdata.DBLoginHistory1,
			testdata.DBLoginHistory2,
		}

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User1ID,
			Limit:  defaultQueryLimit,
			Offset: 0,
		}).Return(dbHistory, nil).Once()

		history, err := store.GetLoginHistory(ctx, testdata.User1ID)

		assert.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, testdata.LoginHistory1ID, history[0].ID)
		assert.Equal(t, testdata.LoginHistory2ID, history[1].ID)
	})

	t.Run("Get login history for user with multiple pages", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		firstPage := make([]db.LoginHistory, 100)
		for i := range firstPage {
			firstPage[i] = db.LoginHistory{
				ID:           uuid.New(),
				UserID:       testdata.User1ID,
				LoginAttempt: testdata.User1.Login,
				Success:      true,
			}
		}

		secondPage := []db.LoginHistory{
			testdata.DBLoginHistory1,
			testdata.DBLoginHistory2,
		}

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User1ID,
			Limit:  100,
			Offset: 0,
		}).Return(firstPage, nil).Once()

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User1ID,
			Limit:  100,
			Offset: 100,
		}).Return(secondPage, nil).Once()

		history, err := store.GetLoginHistory(ctx, testdata.User1ID)

		assert.NoError(t, err)
		assert.Len(t, history, 102)
	})

	t.Run("Get login history for user with no history", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		emptyHistory := []db.LoginHistory{}

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User2ID,
			Limit:  100,
			Offset: 0,
		}).Return(emptyHistory, nil).Once()

		history, err := store.GetLoginHistory(ctx, testdata.User2ID)

		assert.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("Get login history fails on database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbErr := assert.AnError

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User1ID,
			Limit:  100,
			Offset: 0,
		}).Return(nil, dbErr).Once()

		history, err := store.GetLoginHistory(ctx, testdata.User1ID)

		assert.ErrorIs(t, err, dbErr)
		assert.Nil(t, history)
	})

	t.Run("Get login history with exactly 100 records (edge case)", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		exactPage := make([]db.LoginHistory, 100)
		for i := range exactPage {
			exactPage[i] = db.LoginHistory{
				ID:           uuid.New(),
				UserID:       testdata.User1ID,
				LoginAttempt: testdata.User1.Login,
				Success:      true,
			}
		}

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User1ID,
			Limit:  100,
			Offset: 0,
		}).Return(exactPage, nil).Once()

		mockQ.EXPECT().GetLoginHistoryByUser(mock.Anything, db.GetLoginHistoryByUserParams{
			UserID: testdata.User1ID,
			Limit:  100,
			Offset: 100,
		}).Return([]db.LoginHistory{}, nil).Once()

		history, err := store.GetLoginHistory(ctx, testdata.User1ID)

		assert.NoError(t, err)
		assert.Len(t, history, 100)
	})
}
