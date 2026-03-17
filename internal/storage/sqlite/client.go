package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	client *sql.DB
}

func New(ctx context.Context, dbPath string) (*DB, error) {
	// Open Connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// In-memory SQLite creates a separate database per connection.
	// Force a single connection so InitSchema and queries share the same database.
	if dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	// Test Connection
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite. Error: %w", err)
	}

	return &DB{client: db}, nil
}

func (d *DB) Close() error {
	return d.client.Close()
}
