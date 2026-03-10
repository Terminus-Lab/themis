package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	client *sql.DB
}

func New(ctx context.Context, dbPath string) (*DB, error) {
	// Open Connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
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
