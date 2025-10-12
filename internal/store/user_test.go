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
	"github.com/trashscanner/trashscanner_api/internal/store/mocks"
	"github.com/trashscanner/trashscanner_api/internal/testdata"
)

func TestUserCreate(t *testing.T) {
	t.Run("Create user successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		id := uuid.New()
		statsID := uuid.New()
		user := testdata.NewUser

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).Return(db.User{}, pgx.ErrNoRows).Once()

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()
		mockTx.EXPECT().Commit(mock.Anything).Return(nil).Once()

		mockQTx.EXPECT().CreateUser(mock.Anything, db.CreateUserParams{
			Login:          user.Login,
			HashedPassword: user.HashedPassword,
		}).Return(id, nil).Once()

		mockQTx.EXPECT().CreateStats(mock.Anything, id).Return(statsID, nil).Once()

		err := store.CreateUser(ctx, &user)

		assert.NoError(t, err)
		assert.Equal(t, id, user.ID)
	})

	t.Run("User already exists", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
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
		mockConn := mocks.NewConnection(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
		}

		user := testdata.NewUser
		dbErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(db.User{}, dbErr).Once()

		err := store.CreateUser(ctx, &user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check existing user")
	})

	t.Run("Transaction begin fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
		}

		user := testdata.NewUser
		txErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(db.User{}, pgx.ErrNoRows).Once()

		mockConn.EXPECT().Begin(mock.Anything).Return(nil, txErr).Once()

		err := store.CreateUser(ctx, &user)

		assert.ErrorIs(t, err, txErr)
	})

	t.Run("CreateUser in transaction fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		user := testdata.NewUser
		createErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(db.User{}, pgx.ErrNoRows).Once()

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()

		mockQTx.EXPECT().CreateUser(mock.Anything, mock.Anything).
			Return(uuid.Nil, createErr).Once()

		err := store.CreateUser(ctx, &user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user")
	})

	t.Run("CreateStats in transaction fails", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		id := uuid.New()
		user := testdata.NewUser
		statsErr := assert.AnError

		mockQ.EXPECT().GetUserByLogin(mock.Anything, user.Login).
			Return(db.User{}, pgx.ErrNoRows).Once()

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()

		mockQTx.EXPECT().CreateUser(mock.Anything, mock.Anything).
			Return(id, nil).Once()

		mockQTx.EXPECT().CreateStats(mock.Anything, id).
			Return(uuid.Nil, statsErr).Once()

		err := store.CreateUser(ctx, &user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create user stats")
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

	t.Run("Update to new avatar URL", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		user := testdata.User2
		user.Avatar = &testdata.NewAvatarURL

		mockQ.EXPECT().UpdateUserAvatar(mock.Anything, db.UpdateUserAvatarParams{
			ID:     testdata.User2ID,
			Avatar: &testdata.NewAvatarURL,
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

	t.Run("User not found", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)

		store := &pgStore{
			q: mockQ,
		}

		nonExistentID := uuid.New()

		mockQ.EXPECT().DeleteUser(mock.Anything, nonExistentID).
			Return(pgx.ErrNoRows).Once()

		err := store.DeleteUser(ctx, nonExistentID)

		assert.Error(t, err)
		var localErr *errlocal.ErrNotFound
		assert.ErrorAs(t, err, &localErr)
	})
}

func TestExecTx(t *testing.T) {
	t.Run("Execute transaction successfully", func(t *testing.T) {
		ctx := context.Background()

		mockQ := dbMock.NewQuerier(t)
		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			q:    mockQ,
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		dbUser := db.User{
			ID:             testdata.User1ID,
			Login:          testdata.User1.Login,
			HashedPassword: testdata.User1.HashedPassword,
		}

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()
		mockTx.EXPECT().Commit(mock.Anything).Return(nil).Once()

		mockQTx.EXPECT().GetUserByID(mock.Anything, testdata.User1ID).
			Return(dbUser, nil).Once()

		var capturedUser db.User
		err := store.ExecTx(ctx, func(q db.Querier) error {
			user, err := q.GetUserByID(ctx, testdata.User1ID)
			capturedUser = user
			return err
		})

		assert.NoError(t, err)
		assert.Equal(t, testdata.User1ID, capturedUser.ID)
		assert.Equal(t, testdata.User1.Login, capturedUser.Login)
	})

	t.Run("Transaction begin fails", func(t *testing.T) {
		ctx := context.Background()

		mockConn := mocks.NewConnection(t)

		store := &pgStore{
			pool: mockConn,
		}

		txErr := assert.AnError

		mockConn.EXPECT().Begin(mock.Anything).Return(nil, txErr).Once()

		err := store.ExecTx(ctx, func(q db.Querier) error {
			return nil
		})

		assert.ErrorIs(t, err, txErr)
	})

	t.Run("Function returns error and rollback", func(t *testing.T) {
		ctx := context.Background()

		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		fnErr := assert.AnError

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()

		err := store.ExecTx(ctx, func(q db.Querier) error {
			return fnErr
		})

		assert.ErrorIs(t, err, fnErr)
		// Commit не должен вызываться при ошибке
	})

	t.Run("Commit fails", func(t *testing.T) {
		ctx := context.Background()

		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		commitErr := assert.AnError

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()
		mockTx.EXPECT().Commit(mock.Anything).Return(commitErr).Once()

		err := store.ExecTx(ctx, func(q db.Querier) error {
			return nil
		})

		assert.ErrorIs(t, err, commitErr)
	})

	t.Run("Complex transaction with multiple operations", func(t *testing.T) {
		ctx := context.Background()

		mockQTx := dbMock.NewQuerier(t)
		mockConn := mocks.NewConnection(t)
		mockTx := mocks.NewTx(t)

		store := &pgStore{
			pool: mockConn,
			qf: func(tx db.DBTX) db.Querier {
				return mockQTx
			},
		}

		newID := uuid.New()
		statsID := uuid.New()

		mockConn.EXPECT().Begin(mock.Anything).Return(mockTx, nil).Once()
		mockTx.EXPECT().Rollback(mock.Anything).Return(nil).Once()
		mockTx.EXPECT().Commit(mock.Anything).Return(nil).Once()

		// Симулируем создание пользователя и статистики
		mockQTx.EXPECT().CreateUser(mock.Anything, db.CreateUserParams{
			Login:          "new_user",
			HashedPassword: "new_password",
		}).Return(newID, nil).Once()

		mockQTx.EXPECT().CreateStats(mock.Anything, newID).
			Return(statsID, nil).Once()

		var createdUserID uuid.UUID
		err := store.ExecTx(ctx, func(q db.Querier) error {
			id, err := q.CreateUser(ctx, db.CreateUserParams{
				Login:          "new_user",
				HashedPassword: "new_password",
			})
			if err != nil {
				return err
			}
			createdUserID = id

			_, err = q.CreateStats(ctx, id)
			return err
		})

		assert.NoError(t, err)
		assert.Equal(t, newID, createdUserID)
	})
}
