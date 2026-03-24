package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/Terminus-Lab/themis/internal/api"
	"github.com/Terminus-Lab/themis/internal/api/middleware"
	"github.com/Terminus-Lab/themis/internal/env"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/Terminus-Lab/themis/internal/stream"
	"github.com/Terminus-Lab/themis/internal/stream/redis"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	logger := log.Logger

	if err := godotenv.Load(); err != nil {
		logger.Warn().Msg("No .env file found")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg := setup.LoadConfig()

	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to load dependencies")
	}

	handler := api.NewHandler(deps.ConversationEvaluator, deps.Repository, &logger)
	container := restful.NewContainer()
	container.Filter(middleware.Logger)
	container.Filter(middleware.RecoverPanic)
	api.RegisterRoutes(container, handler)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.Handle("/api/", corsHandler.Handler(container))

	port := env.GetString("EVAL_AGENT_API_PORT", "18082")
	addr := fmt.Sprintf(":%s", port)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Conversation streaming consumer (optional)
	convStreamingEnabled := env.GetBool("CONVERSATION_STREAMING_ENABLED", false)
	if convStreamingEnabled {
		startConversationStreamingConsumer(ctx, deps, &logger)
	} else {
		logger.Info().Msg("Streaming mode disabled - API only")
	}

	go func() {
		logger.Info().
			Str("address", addr).
			Bool("conversation_streaming_enabled", convStreamingEnabled).
			Msg("Starting Themis Server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("server failed")
		}
	}()

	<-ctx.Done()
	logger.Info().Msg("shutdown signal received - stopping server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("server forced to shutdown")
	}

	logger.Info().Msg("Themis server stopped")
}

func startConversationStreamingConsumer(ctx context.Context, deps *setup.Dependencies, logger *zerolog.Logger) {
	logger.Info().Msg("conversation streaming enabled - starting Redis consumer")

	streamCfg := &stream.StreamConfig{
		Provider: env.GetString("STREAM_PROVIDER", "redis"),
		RedisConfig: redis.NewRedisStreamConfig(
			env.GetString("REDIS_ADDR", "localhost:6379"),
			env.GetString("REDIS_PASSWORD", ""),
			env.GetString("REDIS_CONVERSATION_STREAM_KEY", "eval-conversations"),
			env.GetString("REDIS_CONVERSATION_GROUP", "eval-conv-group"),
			env.GetString("REDIS_CONSUMER_NAME", env.GetHostname("consumer-1")),
		),
	}

	consumer, err := stream.NewConversationStreamConsumer(ctx, streamCfg, deps.ConversationEvaluator, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create conversation stream consumer")
	}
	if err := consumer.Setup(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to setup conversation consumer")
	}

	go func() {
		logger.Info().
			Str("stream_key", streamCfg.RedisConfig.Stream).
			Str("consumer_group", streamCfg.RedisConfig.Group).
			Msg("conversation streaming consumer started")

		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Err(err).Msg("conversation streaming consumer stopped with error")
		}
	}()
}
