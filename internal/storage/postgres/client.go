package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

type DB struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, config Config) (*DB, error) {
	connString := config.ConnectionString()
	pgPool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database. Error: %w", err)
	}

	return &DB{
		Pool: pgPool,
	}, nil
}

func NewWithBackoff(ctx context.Context, config Config, maxRetries int, log *zerolog.Logger) (*DB, error) {

	connString := config.ConnectionString()

	var pgPool *pgxpool.Pool
	var err error

	for i := range maxRetries {
		// Exponential backoff: 1s, 2s, 4s, 8s, 16s
		backoff := time.Duration(1<<uint(i)) * time.Second

		// avoid sleep on first attempt
		if i > 0 {
			time.Sleep(backoff)
		}

		log.Info().Int("attempts", i+1).Int("max_retries", maxRetries).Msg("Connecting to database")
		pgPool, err = pgxpool.New(ctx, connString)
		if err == nil {
			if err = pgPool.Ping(ctx); err == nil {
				log.Info().Int("attempts_needed", i+1).Msg("Database connected")
				return &DB{Pool: pgPool}, nil
			}
			pgPool.Close()
		}

		log.Warn().Err(err).Int("attempt", i+1).Msg("Connection attempt failed")
	}

	return nil, fmt.Errorf("failed to connect after %d attempts. Error: %w", maxRetries, err)
}

func (c *Config) ConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s", c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)

}

func (d *DB) Close() error {
	d.Pool.Close()
	return nil
}
