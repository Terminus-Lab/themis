package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/Terminus-Lab/themis/internal/stream"
	"github.com/Terminus-Lab/themis/internal/stream/redis"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	logger := log.Logger

	// Load env
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load config and wire dependencies (shared with API, Batch, MCP)
	cfg := setup.LoadConfig()
	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to wire dependencies")
	}

	// Redis stream configuration (streaming-specific)
	streamCfg := &stream.StreamConfig{
		Provider: os.Getenv("STREAM_PROVIDER"),
		RedisConfig: redis.NewRedisStreamConfig(
			os.Getenv("REDIS_ADDR"),
			os.Getenv("REDIS_PASSWORD"),
			"eval-events",
			"eval-group",
			os.Getenv("HOSTNAME"),
		),
	}

	// Create stream consumer with shared executor
	consumer, err := stream.NewStreamConsumer(ctx, streamCfg, deps.Executor, &logger)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create stream consumer")
	}

	// Setup consumer
	err = consumer.Setup(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to setup consumer")
	}

	// Start consumer
	go func() {
		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Err(err).Msg("Consumer stopped with error")
		}
	}()

	// Wait for context to be done
	<-ctx.Done()
	logger.Info().Msg("Shutting down...")

	log.Info().Msg("Eval Agent stopped")
}
