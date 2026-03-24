package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Terminus-Lab/themis/internal/models"
	red "github.com/Terminus-Lab/themis/internal/stream/redis"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	data := flag.String("d", "", "Inline JSON ConversationEvaluationRequest")
	stream := flag.String("stream", "eval-conversations", "Stream name")
	flag.Parse()

	if *data == "" {
		fmt.Fprintln(os.Stderr, "Usage: producer -d '<json>'")
		flag.PrintDefaults()
		os.Exit(1)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	if err := run(*data, *stream); err != nil {
		log.Error().Err(err).Msg("producer failed")
		os.Exit(1)
	}
}

func run(data, stream string) error {
	_ = godotenv.Load()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	ctx := context.Background()
	client, err := red.ConnectRedis(ctx, addr, os.Getenv("REDIS_PASSWORD"), 3)
	if err != nil {
		return err
	}
	defer closeRedisClient(client)

	var req models.ConversationEvaluationRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return err
	}

	id, err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{"payload": data},
	}).Result()
	if err != nil {
		return err
	}

	log.Info().
		Str("stream", stream).
		Str("id", id).
		Str("conversation_id", req.ConversationID).
		Msg("Published successfully!")
	return nil
}

func closeRedisClient(client *redis.Client) {
	if err := client.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close Redis client")
	}
}
