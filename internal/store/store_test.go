package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	dbmocks "github.com/trashscanner/trashscanner_api/internal/database/mocks"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/store"
	storemocks "github.com/trashscanner/trashscanner_api/internal/store/mocks"
)

func TestExecTx_Success(t *testing.T) {
	conn := storemocks.NewConnection(t)
	tx := storemocks.NewTx(t)
	ctx := context.Background()

	conn.EXPECT().Begin(mock.Anything).Return(tx, nil)
	tx.EXPECT().Rollback(mock.Anything).Return(nil)
	tx.EXPECT().Commit(mock.Anything).Return(nil)

	s := store.NewPgStore(nil, func(tx db.DBTX) db.Querier { return nil }, conn)

	var called bool
	err := s.ExecTx(ctx, func(st store.Store) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestExecTx_BeginError(t *testing.T) {
	conn := storemocks.NewConnection(t)
	ctx := context.Background()

	beginErr := errors.New("begin failure")
	conn.EXPECT().Begin(mock.Anything).Return(nil, beginErr)

	s := store.NewPgStore(nil, func(tx db.DBTX) db.Querier { return nil }, conn)

	var called bool
	err := s.ExecTx(ctx, func(st store.Store) error {
		called = true
		return nil
	})

	assert.ErrorIs(t, err, beginErr)
	assert.False(t, called)
}

func TestExecTx_FnError_RollsBack(t *testing.T) {
	conn := storemocks.NewConnection(t)
	tx := storemocks.NewTx(t)
	ctx := context.Background()

	conn.EXPECT().Begin(mock.Anything).Return(tx, nil)
	// on fn error Commit should NOT be called; Rollback happens via defer
	tx.EXPECT().Rollback(mock.Anything).Return(nil)

	expectedErr := errors.New("fn failure")

	s := store.NewPgStore(nil, func(tx db.DBTX) db.Querier { return nil }, conn)

	err := s.ExecTx(ctx, func(st store.Store) error {
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	tx.AssertNotCalled(t, "Commit", mock.Anything)
}

func TestClose_CallsConnectionClose(t *testing.T) {
	conn := storemocks.NewConnection(t)
	conn.EXPECT().Close()

	s := store.NewPgStore(nil, func(tx db.DBTX) db.Querier { return nil }, conn)
	s.Close()
}

func TestWithTx_UsesFactory(t *testing.T) {
	tx := storemocks.NewTx(t)

	var received db.DBTX
	s := store.NewPgStore(nil, func(tx db.DBTX) db.Querier {
		received = tx
		return nil
	}, storemocks.NewConnection(t))

	// When calling WithTx we expect factory to be invoked with provided tx
	_ = s.WithTx(tx)
	require.Equal(t, tx, received)
}

func TestBeginTx_WithTx_Flow(t *testing.T) {
	conn := storemocks.NewConnection(t)
	mockTx := storemocks.NewTx(t)
	ctx := context.Background()

	conn.EXPECT().Begin(mock.Anything).Return(mockTx, nil)

	var txQuerierCalled bool
	txFactory := func(tx db.DBTX) db.Querier {
		if tx == mockTx {
			txQuerierCalled = true
		}
		return nil
	}

	s := store.NewPgStore(nil, txFactory, conn)

	tx, err := s.BeginTx(ctx)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.Equal(t, mockTx, tx)

	txStore := s.WithTx(tx)
	require.NotNil(t, txStore)
	assert.True(t, txQuerierCalled, "txFactory should have been called with the transaction")

	mockTx.EXPECT().Rollback(mock.Anything).Return(nil)
	err = tx.Rollback(ctx)
	require.NoError(t, err)
}

func TestExecTx_WithMethodCalls(t *testing.T) {
	conn := storemocks.NewConnection(t)
	mockTx := storemocks.NewTx(t)
	mockQuerier := dbmocks.NewQuerier(t)
	ctx := context.Background()

	conn.EXPECT().Begin(mock.Anything).Return(mockTx, nil)
	mockTx.EXPECT().Rollback(mock.Anything).Return(nil)
	mockTx.EXPECT().Commit(mock.Anything).Return(nil)

	txFactory := func(tx db.DBTX) db.Querier {
		if tx == mockTx {
			return mockQuerier
		}
		return nil
	}

	s := store.NewPgStore(nil, txFactory, conn)

	var txStoreCalled bool
	err := s.ExecTx(ctx, func(txStore store.Store) error {
		txStoreCalled = true

		require.NotNil(t, txStore)

		return nil
	})

	require.NoError(t, err)
	assert.True(t, txStoreCalled, "ExecTx callback should have been called")
}
