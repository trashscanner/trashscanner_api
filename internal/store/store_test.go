package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	storemocks "github.com/trashscanner/trashscanner_api/internal/store/mocks"
)

func TestPgStoreExecTxSuccess(t *testing.T) {
	conn := storemocks.NewConnection(t)
	tx := storemocks.NewTx(t)
	ctx := context.Background()

	conn.EXPECT().
		Begin(mock.Anything).
		Return(tx, nil)
	tx.EXPECT().
		Rollback(mock.Anything).
		Return(nil)
	tx.EXPECT().
		Commit(mock.Anything).
		Return(nil)

	var receivedDB db.DBTX
	var fnCalled bool

	s := &pgStore{
		pool: conn,
		qf: func(tx db.DBTX) db.Querier {
			receivedDB = tx
			return db.New(tx)
		},
	}

	err := s.ExecTx(ctx, func(q db.Querier) error {
		fnCalled = true
		require.IsType(t, &db.Queries{}, q)
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, fnCalled)
	assert.Equal(t, tx, receivedDB)
}

func TestPgStoreExecTxBeginError(t *testing.T) {
	conn := storemocks.NewConnection(t)
	ctx := context.Background()

	beginErr := errors.New("begin failure")

	conn.EXPECT().
		Begin(mock.Anything).
		Return(pgx.Tx(nil), beginErr)

	s := &pgStore{pool: conn}

	var fnCalled bool
	err := s.ExecTx(ctx, func(db.Querier) error {
		fnCalled = true
		return nil
	})

	assert.ErrorIs(t, err, beginErr)
	assert.False(t, fnCalled)
}

func TestPgStoreExecTxFnError(t *testing.T) {
	conn := storemocks.NewConnection(t)
	tx := storemocks.NewTx(t)
	ctx := context.Background()

	conn.EXPECT().
		Begin(mock.Anything).
		Return(tx, nil)
	tx.EXPECT().
		Rollback(mock.Anything).
		Return(nil)

	expectedErr := errors.New("fn failure")

	s := &pgStore{
		pool: conn,
		qf: func(tx db.DBTX) db.Querier {
			return db.New(tx)
		},
	}

	err := s.ExecTx(ctx, func(db.Querier) error {
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	tx.AssertNotCalled(t, "Commit", mock.Anything)
}

func TestPgStoreClose(t *testing.T) {
	conn := storemocks.NewConnection(t)
	conn.EXPECT().Close()

	s := &pgStore{pool: conn}
	s.Close()
}

func TestPgStoreWithTx(t *testing.T) {
	tx := storemocks.NewTx(t)

	var received db.DBTX

	s := &pgStore{
		qf: func(tx db.DBTX) db.Querier {
			received = tx
			return db.New(tx)
		},
	}

	q := s.WithTx(tx)

	require.NotNil(t, q)
	assert.Equal(t, tx, received)
}
