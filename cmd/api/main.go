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
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	logger := log.Logger

	// Load env
	if err := godotenv.Load(); err != nil {
		logger.Warn().Msg("No .env file found")
	}

	// Context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load Config
	cfg := setup.LoadConfig()

	// Wire dependencies
	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Unable to load dependencies")
	}

	// ===== API =====
	handler := api.NewHandler(deps.Executor, deps.JudgeExecutor, deps.Repository, &logger)
	container := restful.NewContainer()
	container.Filter(middleware.Logger)
	container.Filter(middleware.RecoverPanic)
	api.RegisterRoutes(container, handler)

	// CORS
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	// Serve static files (dashboard)
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	mux.Handle("/api/", corsHandler.Handler(container))

	// Server config
	port := env.GetString("EVAL_AGENT_API_PORT", "18082")
	addr := fmt.Sprintf(":%s", port)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// ===== Streaming Consumer =====
	streamingEnabled := env.GetBool("STREAMING_ENABLED", false)
	if streamingEnabled {
		startStreamingConsumer(ctx, deps, &logger)
	} else {
		logger.Info().Msg("Streaming mode disabled - API only")
	}

	// ===== Start HTTP Server =====
	go func() {
		logger.Info().
			Str("address", addr).
			Bool("streaming_enabled", streamingEnabled).
			Msg("Starting Themis Server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// ===== Graceful Shutdown =====
	<-ctx.Done()
	logger.Info().Msg("Shutdown signal received - stopping server...")

	// Shutdown HTTP server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	logger.Info().Msg("Themis server stopped")
}

func startStreamingConsumer(ctx context.Context, deps *setup.Dependencies, logger *zerolog.Logger) {
	logger.Info().Msg("Streaming mode enabled - starting Redis consumer")

	streamCfg := &stream.StreamConfig{
		Provider: env.GetString("STREAM_PROVIDER", "redis"),
		RedisConfig: redis.NewRedisStreamConfig(
			env.GetString("REDIS_ADDR", "localhost:6379"),
			env.GetString("REDIS_PASSWORD", ""),
			env.GetString("REDIS_STREAM_KEY", "eval-events"),
			env.GetString("REDIS_CONSUMER_GROUP", "eval-group"),
			env.GetString("REDIS_CONSUMER_NAME", env.GetHostname("consumer-1")),
		),
	}

	consumer, err := stream.NewStreamConsumer(ctx, streamCfg, deps.Executor, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create stream consumer")
	}
	if err := consumer.Setup(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to setup consumer")
	}

	go func() {
		logger.Info().
			Str("stream_key", streamCfg.RedisConfig.Stream).
			Str("consumer_group", streamCfg.RedisConfig.Group).
			Str("consumer_name", streamCfg.RedisConfig.ConsumerName).
			Msg("Starting streaming consumer")

		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error().Err(err).Msg("Streaming consumer stopped with error")
		}
	}()
}
