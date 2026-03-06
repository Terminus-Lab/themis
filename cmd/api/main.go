package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/joho/godotenv"
	"github.com/Terminus-Lab/themis/internal/api"
	"github.com/Terminus-Lab/themis/internal/api/middleware"
	"github.com/Terminus-Lab/themis/internal/setup"
	"github.com/rs/cors"
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
		logger.Warn().Msg("No .env file found")
	}

	ctx := context.Background()
	// Load Config
	cfg := setup.LoadConfig()

	// Wire dependencies
	deps, err := setup.Wire(ctx, cfg, &logger)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to load dependencies")
		os.Exit(1)
	}
	// API
	handler := api.NewHandler(deps.Executor, deps.JudgeExecutor, &logger)
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

	// Server
	port := os.Getenv("EVAL_AGENT_API_PORT")
	if port == "" {
		port = "18082"
	}

	addr := fmt.Sprintf(":%s", port)
	logger.Info().Str("address", addr).Msg("Starting Eval Agent API")

	server := http.Server{
		Addr:    addr,
		Handler: corsHandler.Handler(container),
	}

	if err := server.ListenAndServe(); err != nil {
		logger.Fatal().Err(err).Msg("Server failed")
	}
}
