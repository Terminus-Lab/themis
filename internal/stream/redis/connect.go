package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func ConnectRedis(ctx context.Context, addr string, password string, maxRetries int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              0,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	})

	var err error
	for i := range maxRetries {
		if i > 0 {
			backoff := time.Duration(1<<uint(i)) * time.Second
			log.Info().Dur("backoff", backoff).Msg("Waiting before Redis retry")
			time.Sleep(backoff)
		}

		log.Info().Int("attempt", i+1).Int("max_retries", maxRetries).Msg("Connecting to Redis")

		err = client.Ping(ctx).Err()
		if err == nil {
			log.Info().Int("attempts_needed", i+1).Msg("Redis connected")
			return client, nil
		}

		log.Warn().Err(err).Int("attempt", i+1).Msg("Redis ping failed")
	}

	return nil, fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, err)
}
