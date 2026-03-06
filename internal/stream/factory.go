package stream

import (
	"context"
	"fmt"

	"github.com/Terminus-Lab/themis/internal/executor"
	"github.com/Terminus-Lab/themis/internal/stream/redis"
	"github.com/rs/zerolog"
)

type StreamConfig struct {
	Provider    string // redis, kafka, sqs, etc
	RedisConfig *redis.RedisStreamConfig
}

func NewStreamConsumer(
	ctx context.Context,
	cfg *StreamConfig,
	exec *executor.Executor,
	logger *zerolog.Logger,
) (StreamConsumer, error) {

	// If provider is empty, fallback to the default configuration.
	provider := cfg.Provider
	if provider == "" {
		provider = "redis"
	}

	switch provider {
	case "redis":
		if cfg.RedisConfig == nil {
			return nil, fmt.Errorf("redis config required")
		}

		client, err := redis.ConnectRedis(
			ctx,
			cfg.RedisConfig.RedisAddr,
			"", // password from cfg if needed
			5,
		)
		if err != nil {
			return nil, err
		}

		return redis.NewConsumer(
			client,
			cfg.RedisConfig.Stream,
			cfg.RedisConfig.Group,
			cfg.RedisConfig.ConsumerName,
			exec,
			logger,
		), nil

	// Future providers:
	// case "kafka":
	//     return kafka.NewConsumer(...)
	// case "sqs":
	//     return sqs.NewConsumer(...)

	default:
		return nil, fmt.Errorf("unsupported stream provider: %s", cfg.Provider)
	}
}
