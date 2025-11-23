package store

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	dbMock "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/errlocal"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestUserCreate(t *testing.T) {
	t.Run("Create user successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		id := uuid.New()
		user := testdata.NewUser

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).Return(db.User{}, pgx.ErrNoRows).Once()

		mockQ.EXPECT().CreateUser(mock.Anything, db.CreateUserParams{
			Login:          user.Login,
			HashedPassword: user.HashedPassword,
		}).Return(id, nil).Once()

		err := store.CreateUser(ctx, &user)

		assert.NoError(t, err)
		assert.Equal(t, id, user.ID)
	})

	t.Run("User already exists", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.NewUser
		existingUser := db.User{
			ID:    uuid.New(),
			Login: user.Login,
		}

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(existingUser, nil).Once()

		err := store.CreateUser(ctx, &user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("Database error on login check", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.NewUser
		dbErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(db.User{}, dbErr).Once()

		err := store.CreateUser(ctx, &user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check existing user")
	})

	t.Run("CreateUser fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.NewUser
		createErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(db.User{}, pgx.ErrNoRows).Once()

		mockQ.EXPECT().CreateUser(mock.Anything, mock.Anything).
			Return(uuid.Nil, createErr).Once()

		err := store.CreateUser(ctx, &user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user")
	})
}

func TestGetUser(t *testing.T) {
	t.Run("Get user without stats", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbUser := db.User{
			ID:             testdata.User1ID,
			Login:          testdata.User1.Login,
			HashedPassword: testdata.User1.HashedPassword,
			Avatar:         nil,
			Deleted:        false,
		}

		mockQ.EXPECT().GetUserByID(mock.Anything, testdata.User1ID).
			Return(dbUser, nil).Once()

		user, err := store.GetUser(ctx, testdata.User1ID, false)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testdata.User1ID, user.ID)
		assert.Equal(t, testdata.User1.Login, user.Login)
		assert.Nil(t, user.Stat)
	})

	t.Run("Get user with stats", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbUser := db.User{
			ID:             testdata.User1ID,
			Login:          testdata.User1.Login,
			HashedPassword: testdata.User1.HashedPassword,
			Avatar:         &testdata.AvatarURL,
			Deleted:        false,
		}

		dbStats := db.Stat{
			ID:     testdata.Stats1ID,
			UserID: testdata.User1ID,
			Status: string(testdata.Stats1.Status),
			Rating: int32(testdata.Stats1.Rating),
		}

		mockQ.EXPECT().GetUserByID(mock.Anything, testdata.User1ID).
			Return(dbUser, nil).Once()

		mockQ.EXPECT().GetStatsByUserID(mock.Anything, testdata.User1ID).
			Return(dbStats, nil).Once()

		user, err := store.GetUser(ctx, testdata.User1ID, true)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testdata.User1ID, user.ID)
		assert.Equal(t, testdata.User1.Login, user.Login)
		assert.NotNil(t, user.Stat)
		assert.Equal(t, testdata.Stats1.Status, user.Stat.Status)
		assert.Equal(t, testdata.Stats1.Rating, user.Stat.Rating)
	})

	t.Run("User not found", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		nonExistentID := uuid.New()

		mockQ.EXPECT().GetUserByID(mock.Anything, nonExistentID).
			Return(db.User{}, pgx.ErrNoRows).Once()

		user, err := store.GetUser(ctx, nonExistentID, false)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("Stats fetch fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbUser := db.User{
			ID:             testdata.User2ID,
			Login:          testdata.User2.Login,
			HashedPassword: testdata.User2.HashedPassword,
			Deleted:        false,
		}

		statsErr := assert.AnError

		mockQ.EXPECT().GetUserByID(mock.Anything, testdata.User2ID).
			Return(dbUser, nil).Once()

		mockQ.EXPECT().GetStatsByUserID(mock.Anything, testdata.User2ID).
			Return(db.Stat{}, statsErr).Once()

		user, err := store.GetUser(ctx, testdata.User2ID, true)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "failed to get user stats")
	})
}

func TestGetUserByLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		dbUser := db.User{
			ID:             testdata.User1ID,
			Login:          testdata.User1.Login,
			HashedPassword: testdata.User1.HashedPassword,
			Avatar:         nil,
			Deleted:        false,
		}

		mockQ.EXPECT().GetUserByLogin(mock.Anything, testdata.User1.Login).
			Return(dbUser, nil).Once()

		user, err := store.GetUserByLogin(ctx, testdata.User1.Login)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testdata.User1ID, user.ID)
		assert.Equal(t, testdata.User1.Login, user.Login)
		assert.Equal(t, testdata.User1.HashedPassword, user.HashedPassword)
	})

	t.Run("not found", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		login := "missing_user"

		mockQ.EXPECT().GetUserByLogin(mock.Anything, login).
			Return(db.User{}, pgx.ErrNoRows).Once()

		user, err := store.GetUserByLogin(ctx, login)

		assert.Error(t, err)
		assert.Nil(t, user)
		var notFoundErr *errlocal.ErrNotFound
		assert.ErrorAs(t, err, &notFoundErr)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("database error", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		login := "db_error_user"
		dbErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, login).
			Return(db.User{}, dbErr).Once()

		user, err := store.GetUserByLogin(ctx, login)

		assert.Error(t, err)
		assert.Nil(t, user)
		var internalErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &internalErr)
		assert.Contains(t, internalErr.System(), dbErr.Error())
	})
}

func TestUpdateUserPass(t *testing.T) {
	t.Run("Update password successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		mockQ.EXPECT().UpdateUserPassword(mock.Anything, db.UpdateUserPasswordParams{
			ID:             testdata.User1ID,
			HashedPassword: testdata.NewHashedPassword,
		}).Return(nil).Once()

		err := store.UpdateUserPass(ctx, testdata.User1ID, testdata.NewHashedPassword)

		assert.NoError(t, err)
	})

	t.Run("Update password fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		updateErr := assert.AnError

		mockQ.EXPECT().UpdateUserPassword(mock.Anything, mock.Anything).
			Return(updateErr).Once()

		err := store.UpdateUserPass(ctx, testdata.User2ID, testdata.NewHashedPassword)

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), updateErr.Error())
	})
}

func TestUpdateAvatar(t *testing.T) {
	t.Run("Update avatar successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.User1
		user.Avatar = &testdata.AvatarURL

		mockQ.EXPECT().UpdateUserAvatar(mock.Anything, db.UpdateUserAvatarParams{
			ID:     testdata.User1ID,
			Avatar: &testdata.AvatarURL,
		}).Return(nil).Once()

		err := store.UpdateAvatar(ctx, &user)

		assert.NoError(t, err)
	})

	t.Run("Update avatar fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		updateErr := assert.AnError

		mockQ.EXPECT().UpdateUserAvatar(mock.Anything, mock.Anything).
			Return(updateErr).Once()

		err := store.UpdateAvatar(ctx, &testdata.User1)

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), updateErr.Error())
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("Update user successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.User1
		user.Name = "Updated Name"

		mockQ.EXPECT().UpdateUser(mock.Anything, db.UpdateUserParams{
			ID:   testdata.User1ID,
			Name: "Updated Name",
		}).Return(nil).Once()

		err := store.UpdateUser(ctx, &user)

		assert.NoError(t, err)
	})

	t.Run("Update user fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.User2
		user.Name = "New Name"
		updateErr := assert.AnError

		mockQ.EXPECT().UpdateUser(mock.Anything, mock.Anything).
			Return(updateErr).Once()

		err := store.UpdateUser(ctx, &user)

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), updateErr.Error())
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("Delete user successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		mockQ.EXPECT().DeleteUser(mock.Anything, testdata.User1ID).
			Return(nil).Once()

		err := store.DeleteUser(ctx, testdata.User1ID)

		assert.NoError(t, err)
	})

	t.Run("Delete user fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		deleteErr := assert.AnError

		mockQ.EXPECT().DeleteUser(mock.Anything, testdata.User2ID).
			Return(deleteErr).Once()

		err := store.DeleteUser(ctx, testdata.User2ID)

		assert.Error(t, err)
		var localErr *errlocal.ErrInternal
		assert.ErrorAs(t, err, &localErr)
		assert.Contains(t, localErr.System(), deleteErr.Error())
	})
}
