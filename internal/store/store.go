package store

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/trashscanner/trashscanner_api/internal/config"
	"github.com/trashscanner/trashscanner_api/internal/database/sqlc/db"
	"github.com/trashscanner/trashscanner_api/internal/models"
)

const (
	connTimeout       = time.Second * 5
	defaultQueryLimit = 100
)

type Store interface {
	StartPrediction(ctx context.Context, userID uuid.UUID, scanURL string) (*models.Prediction, error)
	CompletePrediction(ctx context.Context, id uuid.UUID, result models.PredictionResult, err error) error
	GetPrediction(ctx context.Context, id uuid.UUID) (*models.Prediction, error)
	GetPredictionsByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.Prediction, error)

	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, id uuid.UUID, withStats bool) (*models.User, error)
	GetUserByLogin(ctx context.Context, login string) (*models.User, error)
	UpdateUserPass(ctx context.Context, id uuid.UUID, newHashedPass string) error
	UpdateAvatar(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error

	InsertRefreshToken(ctx context.Context, refreshToken *models.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	UpdateStats(ctx context.Context, stat *models.Stat) error

	InsertLoginHistory(ctx context.Context, loginHistory *models.LoginHistory) error
	GetLoginHistory(ctx context.Context, userID uuid.UUID) ([]models.LoginHistory, error)

	Close()
	Conn() *pgxpool.Pool
	WithTx(tx pgx.Tx) Store
	BeginTx(ctx context.Context) (pgx.Tx, error)
	ExecTx(ctx context.Context, fn func(Store) error) error
}

type Connection interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Close()
}

type QuerierFactory func(tx db.DBTX) db.Querier

type pgStore struct {
	q    db.Querier
	qf   QuerierFactory
	pool Connection
}

func CreatePgStore(conf config.Config) (Store, error) {
	dsn := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%s", conf.DB.Host, conf.DB.Port),
		User:   url.UserPassword(conf.DB.User, conf.DB.Password),
		Path:   "/" + conf.DB.Name,
	}

	ctx, cancel := context.WithTimeout(context.Background(), connTimeout)
	defer cancel()

	conn, err := pgxpool.New(ctx, dsn.String())
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, err
	}

	return &pgStore{
		pool: conn,
		q:    db.New(conn),
		qf:   func(tx db.DBTX) db.Querier { return db.New(tx) },
	}, nil
}

func NewPgStore(q db.Querier, qf QuerierFactory, pool Connection) Store {
	return &pgStore{
		q:    q,
		qf:   qf,
		pool: pool,
	}
}

func (s *pgStore) Conn() *pgxpool.Pool {
	return s.pool.(*pgxpool.Pool)
}

func (s *pgStore) Close() {
	s.pool.Close()
}

func (s *pgStore) WithTx(tx pgx.Tx) Store {
	return &pgStore{
		q:    s.qf(tx),
		qf:   s.qf,
		pool: s.pool,
	}
}

func (s *pgStore) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

func (s *pgStore) ExecTx(ctx context.Context, fn func(Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(s.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
