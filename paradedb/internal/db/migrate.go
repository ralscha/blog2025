package db

import (
	"context"
	"database/sql"
	"fmt"

	dbmigrations "paradedb-demo/db/migrations"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

func RunMigrations(ctx context.Context, db *sql.DB) error {
	provider, err := newMigrationProvider(db)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func newMigrationProvider(db *sql.DB) (*goose.Provider, error) {
	migrationLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("create migration locker: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, dbmigrations.FS, goose.WithSessionLocker(migrationLocker))
	if err != nil {
		return nil, err
	}

	return provider, nil
}
