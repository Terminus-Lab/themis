package stream

import (
	"context"
	"fmt"

	"github.com/Terminus-Lab/themis/internal/executor"
	streamredis "github.com/Terminus-Lab/themis/internal/stream/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type StreamConfig struct {
	Provider    string // redis, kafka, sqs, etc
	RedisConfig *streamredis.RedisStreamConfig
}

func NewConversationStreamConsumer(
	ctx context.Context,
	cfg *StreamConfig,
	eval *executor.ConversationEvaluator,
	logger *zerolog.Logger,
) (StreamConsumer, error) {
	client, err := connectRedis(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return streamredis.NewConversationConsumer(client, cfg.RedisConfig.Stream, cfg.RedisConfig.Group, cfg.RedisConfig.ConsumerName, eval, logger), nil
}

func connectRedis(ctx context.Context, cfg *StreamConfig) (*goredis.Client, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = "redis"
	}
	switch provider {
	case "redis":
		if cfg.RedisConfig == nil {
			return nil, fmt.Errorf("redis config required")
		}
		return streamredis.ConnectRedis(ctx, cfg.RedisConfig.RedisAddr, cfg.RedisConfig.RedisPassword, 5)
	default:
		return nil, fmt.Errorf("unsupported stream provider: %s", provider)
	}
}
