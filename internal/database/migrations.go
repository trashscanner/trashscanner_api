package database

import (
	"errors"
	"fmt"

	m "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/trashscanner/trashscanner_api/internal/config"
)

func setupMigrations(pool *pgxpool.Pool, conf config.Config) (*m.Migrate, error) {
	sqlDB := stdlib.OpenDBFromPool(pool)

	driver, err := pgx.WithInstance(sqlDB, &pgx.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	migrationsPath := fmt.Sprintf("file://%s", conf.DB.MigrationsPath)

	migrate, err := m.NewWithDatabaseInstance(migrationsPath, conf.DB.Name, driver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, dirty, err := migrate.Version()
	if err != nil && !errors.Is(err, m.ErrNilVersion) {
		return nil, fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		return nil, fmt.Errorf("database is in dirty state (version %d), please fix manually", version)
	}

	return migrate, nil
}

func RunMigrations(pool *pgxpool.Pool, conf config.Config) error {
	migrate, err := setupMigrations(pool, conf)
	if err != nil {
		return fmt.Errorf("failed to setup migrations: %w", err)
	}

	if err := migrate.Up(); err != nil {
		if errors.Is(err, m.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func DownMigrations(pool *pgxpool.Pool, conf config.Config) error {
	migrate, err := setupMigrations(pool, conf)
	if err != nil {
		return fmt.Errorf("failed to setup migrations: %w", err)
	}
	defer func() {
		_, _ = migrate.Close()
	}()

	if err := migrate.Down(); err != nil {
		if errors.Is(err, m.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to down migrations: %w", err)
	}

	return nil
}
